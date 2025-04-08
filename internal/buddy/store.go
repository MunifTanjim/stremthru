package buddy

import (
	"regexp"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/magnet_cache"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/store"
)

var Buddy = NewAPIClient(&APIClientConfig{
	BaseURL: config.BuddyURL,
})

var buddyLog = logger.Scoped("buddy")

var Peer = peer.NewAPIClient(&peer.APIClientConfig{
	BaseURL: config.PeerURL,
	APIKey:  config.PeerAuthToken,
})

var peerLog = logger.Scoped("buddy:upstream")

func TrackMagnet(s store.Store, hash string, files []store.MagnetFile, cacheMiss bool, storeToken string) {
	mcFiles := magnet_cache.Files{}
	if !cacheMiss {
		for _, f := range files {
			mcFiles = append(mcFiles, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
		}
	}
	magnet_cache.Touch(s.GetName().Code(), hash, mcFiles)

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:     s.GetName(),
			Hash:      hash,
			Files:     files,
			CacheMiss: cacheMiss,
		}
		start := time.Now()
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			buddyLog.Error("failed to track magnet cache", "store", s.GetName(), "hash", hash, "error", core.PackError(err), "duration", time.Since(start))
		} else {
			buddyLog.Info("track magnet cache", "store", s.GetName(), "hash", hash, "duration", time.Since(start))
		}
	}

	if config.HasPeer && config.PeerAuthToken != "" {
		params := &peer.TrackMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
			Hash:       hash,
			Files:      files,
			IsMiss:     cacheMiss,
		}
		go func() {
			start := time.Now()
			if _, err := Peer.TrackMagnet(params); err != nil {
				peerLog.Error("failed to track magnet cache", "store", s.GetName(), "hash", hash, "error", core.PackError(err), "duration", time.Since(start))
			} else {
				peerLog.Info("track magnet cache", "store", s.GetName(), "hash", hash, "duration", time.Since(start))
			}
		}()
	}
}

func BulkTrackMagnet(s store.Store, filesByHash map[string][]store.MagnetFile, storeToken string) {
	mcFilesByHash := map[string]magnet_cache.Files{}
	for hash, files := range filesByHash {
		mcFiles := magnet_cache.Files{}
		for _, f := range files {
			mcFiles = append(mcFiles, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
		}
		mcFilesByHash[hash] = mcFiles
	}
	magnet_cache.BulkTouch(s.GetName().Code(), mcFilesByHash)

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:       s.GetName(),
			FilesByHash: filesByHash,
		}
		start := time.Now()
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			buddyLog.Error("failed to bulk track magnet cache", "store", s.GetName(), "error", core.PackError(err), "duration", time.Since(start))
		} else {
			buddyLog.Info("bulk track magnet cache", "store", s.GetName(), "hash_count", len(filesByHash), "duration", time.Since(start))
		}
	}

	if config.HasPeer && config.PeerAuthToken != "" {
		params := &peer.TrackMagnetParams{
			StoreName:   s.GetName(),
			StoreToken:  storeToken,
			FilesByHash: filesByHash,
		}
		go func() {
			start := time.Now()
			if _, err := Peer.TrackMagnet(params); err != nil {
				peerLog.Error("failed to bulk track magnet cache", "store", s.GetName(), "error", core.PackError(err), "duration", time.Since(start))
			} else {
				peerLog.Info("bulk track magnet cache", "store", s.GetName(), "hash_count", len(filesByHash), "duration", time.Since(start))
			}
		}()
	}
}

func CheckMagnet(s store.Store, hashes []string, storeToken string, clientIp string, sid string) (*store.CheckMagnetData, error) {
	if matched, err := regexp.MatchString("^tt[0-9]+(:[0-9]{1,2}:[0-9]{1,3})?$", sid); err != nil || !matched {
		sid = ""
	}

	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}

	mcs, err := magnet_cache.GetByHashes(s.GetName().Code(), hashes, sid)
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
		params.SId = sid
		start := time.Now()
		res, err := Buddy.CheckMagnetCache(params)
		duration := time.Since(start)
		if err != nil {
			buddyLog.Error("failed to check magnet", "store", s.GetName(), "error", core.PackError(err), "duration", duration)
			return data, nil
		} else {
			buddyLog.Info("check magnet", "store", s.GetName(), "hash_count", len(staleOrMissingHashes), "duration", duration)
			filesByHash := map[string]magnet_cache.Files{}
			for _, item := range res.Data.Items {
				res_item := store.CheckMagnetDataItem{
					Hash:   item.Hash,
					Magnet: item.Magnet,
					Status: item.Status,
				}
				res_files := []store.MagnetFile{}
				files := magnet_cache.Files{}
				if item.Status == store.MagnetStatusCached {
					seenByName := map[string]bool{}
					for _, f := range item.Files {
						if _, seen := seenByName[f.Name]; seen {
							buddyLog.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
							continue
						}
						seenByName[f.Name] = true
						res_files = append(res_files, store.MagnetFile{Idx: f.Idx, Name: f.Name, Size: f.Size})
						files = append(files, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size, SId: f.SId})
					}
				}
				res_item.Files = res_files
				data.Items = append(data.Items, res_item)
				filesByHash[item.Hash] = files
			}
			go magnet_cache.BulkTouch(s.GetName().Code(), filesByHash)
			return data, nil
		}
	}

	if config.HasPeer {
		if Peer.IsHaltedCheckMagnet() {
			return data, nil
		}
		params := &peer.CheckMagnetParams{
			StoreName:  s.GetName(),
			StoreToken: storeToken,
		}
		params.Magnets = staleOrMissingHashes
		params.ClientIP = clientIp
		params.SId = sid
		start := time.Now()
		res, err := Peer.CheckMagnet(params)
		duration := time.Since(start)
		if duration.Seconds() > 10 {
			Peer.HaltCheckMagnet()
		}
		if err != nil {
			peerLog.Error("failed to check magnet", "store", s.GetName(), "error", core.PackError(err), "duration", duration)
			return data, nil
		} else {
			peerLog.Info("check magnet", "store", s.GetName(), "hash_count", len(staleOrMissingHashes), "duration", duration)
			filesByHash := map[string]magnet_cache.Files{}
			for _, item := range res.Data.Items {
				files := magnet_cache.Files{}
				if item.Status == store.MagnetStatusCached {
					seenByName := map[string]bool{}
					for _, f := range item.Files {
						if _, seen := seenByName[f.Name]; seen {
							peerLog.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
							continue
						}
						seenByName[f.Name] = true
						files = append(files, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
					}
				}
				filesByHash[item.Hash] = files
				data.Items = append(data.Items, item)
			}
			go magnet_cache.BulkTouch(s.GetName().Code(), filesByHash)
			return data, nil
		}
	}

	return data, nil
}
