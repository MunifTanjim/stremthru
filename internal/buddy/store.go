package buddy

import (
	"log"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/magnet_cache"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/store"
)

var Buddy = NewAPIClient(&APIClientConfig{
	BaseURL: config.BuddyURL,
})

var Peer = peer.NewAPIClient(&peer.APIClientConfig{
	BaseURL: config.PeerURL,
	APIKey:  config.PeerAuthToken,
})

func TrackMagnet(s store.Store, hash string, files []store.MagnetFile, cacheMiss bool, storeToken string) {
	mcFiles := magnet_cache.Files{}
	if !cacheMiss {
		for _, f := range files {
			mcFiles = append(mcFiles, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
		}
	}
	if err := magnet_cache.Touch(s.GetName().Code(), hash, mcFiles); err != nil {
		log.Printf("[buddy] failed to update local cache: %v\n", err)
	}

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:     s.GetName(),
			Hash:      hash,
			Files:     files,
			CacheMiss: cacheMiss,
		}
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			log.Printf("[buddy] failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
		}
	}

	if config.HasPeer {
		params := &peer.TrackMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
			Hash:       hash,
			Files:      files,
			IsMiss:     cacheMiss,
		}
		if _, err := Peer.TrackMagnet(params); err != nil {
			log.Printf("[buddy:upstream] failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
		}
	}
}

func CheckMagnet(s store.Store, hashes []string, storeToken string, clientIp string) (*store.CheckMagnetData, error) {
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}

	mcs, err := magnet_cache.GetByHashes(s.GetName().Code(), hashes)
	if err != nil {
		return nil, err
	}
	mcByHash := map[string]magnet_cache.MagnetCache{}
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
			Store:    s.GetName(),
			Hashes:   staleOrMissingHashes,
			ClientIP: clientIp,
		}
		res, err := Buddy.CheckMagnetCache(params)
		if err != nil {
			log.Printf("[buddy] failed to check magnet: %v\n", err)
		} else {
			filesByHash := map[string]magnet_cache.Files{}
			for _, item := range res.Data.Items {
				files := magnet_cache.Files{}
				if item.Status == store.MagnetStatusCached {
					for _, f := range item.Files {
						files = append(files, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
					}
				}
				filesByHash[item.Hash] = files
				data.Items = append(data.Items, item)
			}
			err = magnet_cache.BulkTouch(s.GetName().Code(), filesByHash)
			if err != nil {
				log.Printf("[buddy] failed to update local cache: %v\n", err)
			}
			return data, nil
		}
	}

	if config.HasPeer {
		params := &peer.CheckMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
		}
		params.Magnets = hashes
		res, err := Peer.CheckMagnet(params)
		if err != nil {
			log.Printf("[buddy:upstream] failed to check magnet: %v\n", err)
		} else {
			filesByHash := map[string]magnet_cache.Files{}
			for _, item := range res.Data.Items {
				files := magnet_cache.Files{}
				if item.Status == store.MagnetStatusCached {
					for _, f := range item.Files {
						files = append(files, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
					}
				}
				filesByHash[item.Hash] = files
				data.Items = append(data.Items, item)
			}
			err = magnet_cache.BulkTouch(s.GetName().Code(), filesByHash)
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
