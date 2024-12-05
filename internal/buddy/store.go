package buddy

import (
	"log"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/upstream"
	"github.com/MunifTanjim/stremthru/store"
)

var Buddy = NewAPIClient(&APIClientConfig{
	BaseURL: config.BuddyURL,
	APIKey:  config.BuddyAuthToken,
})

var Upstream = upstream.NewAPIClient(&upstream.APIClientConfig{
	BaseURL: config.UpstreamURL,
	APIKey:  config.UpstreamAuthToken,
})

func TrackMagnet(s store.Store, hash string, files []store.MagnetFile, cacheMiss bool, buddyToken string, storeToken string) {
	mcFiles := db.MagnetCacheFiles{}
	if !cacheMiss {
		for _, f := range files {
			mcFiles = append(mcFiles, db.MagnetCacheFile{Idx: f.Idx, Name: f.Name, Size: f.Size})
		}
	}
	if err := db.TouchMagnetCache(s.GetName().Code(), hash, mcFiles); err != nil {
		log.Printf("[buddy] failed to update local cache: %v\n", err)
	}

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:     s.GetName(),
			Hash:      hash,
			Files:     files,
			CacheMiss: cacheMiss,
		}
		params.APIKey = buddyToken
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			log.Printf("[buddy] failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
		}
	}

	if config.HasUpstream {
		params := &upstream.TrackMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
			Hash:       hash,
			Files:      files,
			IsMiss:     cacheMiss,
		}
		if _, err := Upstream.TrackMagnet(params); err != nil {
			log.Printf("[buddy:upstream] failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
		}
	}
}

func CheckMagnet(s store.Store, hashes []string, buddyToken string, storeToken string) (*store.CheckMagnetData, error) {
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}

	mcs, err := db.GetMagnetCaches(s.GetName().Code(), hashes)
	if err != nil {
		return nil, err
	}
	mcByHash := map[string]db.MagnetCache{}
	for _, mc := range mcs {
		mcByHash[mc.Hash] = mc
	}

	magnetByHash := map[string]core.MagnetLink{}

	staleOrMissingHashes := []string{}
	for _, hash := range hashes {
		magnet, err := core.ParseMagnetLink(hash)
		if err != nil {
			continue
		}
		magnetByHash[magnet.Hash] = magnet
		if mc, ok := mcByHash[magnet.Hash]; ok && !mc.IsStale() {
			item := store.CheckMagnetDataItem{
				Hash:   magnet.Hash,
				Magnet: magnet.Link,
				Status: store.MagnetStatusUnknown,
				Files:  []store.MagnetFile{},
			}
			if mc.IsCached {
				item.Status = store.MagnetStatusCached
				item.Files = mc.Files.ToStoreMagnetFile()
			}
			data.Items = append(data.Items, item)
		} else {
			staleOrMissingHashes = append(staleOrMissingHashes, magnet.Hash)
		}
	}

	if len(staleOrMissingHashes) == 0 {
		return data, nil
	}

	if config.HasBuddy {
		params := &CheckMagnetCacheParams{
			Store:  s.GetName(),
			Hashes: staleOrMissingHashes,
		}
		params.APIKey = buddyToken
		res, err := Buddy.CheckMagnetCache(params)
		if err != nil {
			log.Printf("[buddy] failed to check magnet: %v\n", err)
		} else {
			filesByHash := map[string]db.MagnetCacheFiles{}
			for _, item := range res.Data.Items {
				files := db.MagnetCacheFiles{}
				if item.Status == store.MagnetStatusCached {
					for _, f := range item.Files {
						files = append(files, db.MagnetCacheFile{Idx: f.Idx, Name: f.Name, Size: f.Size})
					}
				}
				filesByHash[item.Hash] = files
				data.Items = append(data.Items, item)
			}
			err = db.TouchMagnetCaches(s.GetName().Code(), filesByHash)
			if err != nil {
				log.Printf("[buddy] failed to update local cache: %v\n", err)
			}
			return data, nil
		}
	}

	if config.HasUpstream {
		params := &upstream.CheckMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
		}
		params.Magnets = hashes
		res, err := Upstream.CheckMagnet(params)
		if err != nil {
			log.Printf("[buddy:upstream] failed to check magnet: %v\n", err)
		} else {
			filesByHash := map[string]db.MagnetCacheFiles{}
			for _, item := range res.Data.Items {
				files := db.MagnetCacheFiles{}
				if item.Status == store.MagnetStatusCached {
					for _, f := range item.Files {
						files = append(files, db.MagnetCacheFile{Idx: f.Idx, Name: f.Name, Size: f.Size})
					}
				}
				filesByHash[item.Hash] = files
				data.Items = append(data.Items, item)
			}
			err = db.TouchMagnetCaches(s.GetName().Code(), filesByHash)
			if err != nil {
				log.Printf("[buddy:upstream] failed to update local cache: %v\n", err)
			}
			return data, nil
		}
	}

	for _, hash := range staleOrMissingHashes {
		magnet := magnetByHash[hash]
		item := store.CheckMagnetDataItem{
			Hash:   magnet.Hash,
			Magnet: magnet.Link,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}
		data.Items = append(data.Items, item)
	}
	return data, nil
}

var isValidTokenCache = func() cache.Cache[bool] {
	return cache.NewCache[bool](&cache.CacheConfig{
		Name:     "buddy:isValidToken",
		Lifetime: 10 * time.Minute,
	})
}()

func IsValidToken(token string) (bool, error) {
	if token == "" || !config.HasBuddy {
		return false, nil
	}

	isValid := false
	if isValidTokenCache.Get(token, &isValid) {
		return isValid, nil
	}

	if res, err := Buddy.CheckAuth(&CheckAuthParams{Token: token}); err != nil {
		if res.StatusCode != 401 {
			return false, err
		}
		isValid = false
	} else {
		isValid = true
	}

	if err := isValidTokenCache.Add(token, isValid); err != nil {
		log.Printf("[buddy] failed to cache valid token check: %v\n", err)
	}
	return isValid, nil
}
