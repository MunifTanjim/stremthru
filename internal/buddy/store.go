package buddy

import (
	"log"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/store"
)

var Client = NewAPIClient(&APIClientConfig{
	BaseURL: config.BuddyURL,
	APIKey:  config.BuddyAuthToken,
})

func TrackMagnet(s store.Store, hash string, files []store.MagnetFile, cacheMiss bool) {
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
		if _, err := Client.TrackMagnetCache(&TrackMagnetCacheParams{
			Store:     s.GetName(),
			Hash:      hash,
			Files:     files,
			CacheMiss: cacheMiss,
		}); err != nil {
			log.Printf("failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
		}
	}
}

func CheckMagnet(s store.Store, hashes []string) (*store.CheckMagnetData, error) {
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

	if !config.HasBuddy {
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

	res, err := Client.CheckMagnetCache(&CheckMagnetCacheParams{
		Store:  s.GetName(),
		Hashes: staleOrMissingHashes,
	})
	if err != nil {
		log.Printf("[buddy] failed to check magnet: %v\n", err)

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
