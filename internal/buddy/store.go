package buddy

import (
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/magnet_cache"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
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

func TrackMagnet(s store.Store, hash string, name string, size int64, private bool, files []store.MagnetFile, tInfoCategory torrent_info.TorrentInfoCategory, cacheMiss bool, storeToken string) {
	storeCode := s.GetName().Code()
	tInfoSource := torrent_info.TorrentInfoSource(storeCode)
	tsFiles := torrent_stream.Files{}
	for _, f := range files {
		source := f.Source
		if source == "" {
			source = string(tInfoSource)
		}
		tsFiles = append(tsFiles, torrent_stream.File{
			Idx:       f.Idx,
			Path:      f.Path,
			Name:      f.Name,
			Size:      f.Size,
			Source:    source,
			VideoHash: f.VideoHash,
			MediaInfo: f.MediaInfo,
		})
	}
	magnet_cache.Touch(s.GetName().Code(), hash, tsFiles, !cacheMiss, true)
	go torrent_info.Upsert([]torrent_info.TorrentInfoInsertData{{
		Hash:         hash,
		TorrentTitle: name,
		Size:         size,
		Source:       tInfoSource,
		Files:        tsFiles,
		Private:      private,
	}}, tInfoCategory, storeCode != store.StoreCodeRealDebrid)

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:     s.GetName(),
			Hash:      hash,
			Files:     files,
			CacheMiss: cacheMiss,
		}
		start := time.Now()
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			buddyLog.Error("failed to track magnet cache", "store.name", s.GetName(), "hash", hash, "error", core.PackError(err), "duration", time.Since(start))
		} else {
			buddyLog.Info("track magnet cache", "store.name", s.GetName(), "hash", hash, "duration", time.Since(start))
		}
	}

	if config.HasPeer && config.PeerAuthToken != "" {
		params := &peer.TrackMagnetParams{
			StoreName:           s.GetName(),
			StoreToken:          storeToken,
			TorrentInfoCategory: tInfoCategory,
			TorrentInfos: []torrent_info.TorrentInfoInsertData{
				{
					Hash:         hash,
					TorrentTitle: name,
					Source:       tInfoSource,
					Size:         size,
					Files:        tsFiles,
					Private:      private,
				},
			},
		}
		go func() {
			start := time.Now()
			if _, err := Peer.TrackMagnet(params); err != nil {
				peerLog.Error("failed to track magnet cache", "store.name", s.GetName(), "hash", hash, "error", core.PackError(err), "duration", time.Since(start))
			} else {
				peerLog.Info("track magnet cache", "store.name", s.GetName(), "hash", hash, "duration", time.Since(start))
			}
		}()
	}
}

type TorrentInfoInput = torrent_info.TorrentInfoInsertData

func BulkTrackMagnet(s store.Store, tInfos []TorrentInfoInput, cached map[string]bool, tInfoCategory torrent_info.TorrentInfoCategory, storeToken string) {
	if len(tInfos) == 0 && len(cached) == 0 {
		return
	}

	storeCode := s.GetName().Code()
	tInfoSource := torrent_info.TorrentInfoSource(storeCode)
	filesByHash := map[string]torrent_stream.Files{}
	for i := range tInfos {
		tInfo := &tInfos[i]
		if tInfo.Source == "" {
			tInfo.Source = tInfoSource
		}
		filesByHash[tInfo.Hash] = tInfo.Files
	}
	magnet_cache.BulkTouch(s.GetName().Code(), filesByHash, cached, true)
	go torrent_info.Upsert(tInfos, tInfoCategory, storeCode != store.StoreCodeRealDebrid)

	if config.HasBuddy {
		params := &TrackMagnetCacheParams{
			Store:       s.GetName(),
			FilesByHash: filesByHash,
		}
		start := time.Now()
		if _, err := Buddy.TrackMagnetCache(params); err != nil {
			buddyLog.Error("failed to bulk track magnet cache", "error", core.PackError(err), "hash_count", len(tInfos), "store.name", s.GetName(), "duration", time.Since(start))
		} else {
			buddyLog.Info("bulk track magnet cache", "hash_count", len(tInfos), "store.name", s.GetName(), "duration", time.Since(start))
		}
	}

	if config.HasPeer && config.PeerAuthToken != "" {
		params := &peer.TrackMagnetParams{
			StoreName:           s.GetName(),
			StoreToken:          storeToken,
			TorrentInfoCategory: tInfoCategory,
			TorrentInfos:        tInfos,
			Cached:              cached,
		}
		go func() {
			start := time.Now()
			if _, err := Peer.TrackMagnet(params); err != nil {
				peerLog.Error("failed to bulk track magnet cache", "error", core.PackError(err), "hash_count", len(tInfos), "store.name", s.GetName(), "duration", time.Since(start))
			} else {
				peerLog.Info("bulk track magnet cache", "hash_count", len(tInfos), "store.name", s.GetName(), "duration", time.Since(start))
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
				item.Files = mc.Files.ToStoreMagnetFiles(magnet.Hash)
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
			buddyLog.Error("failed to check magnet", "store.name", s.GetName(), "error", core.PackError(err), "duration", duration)
			return data, nil
		} else {
			buddyLog.Info("check magnet", "store.name", s.GetName(), "hash_count", len(staleOrMissingHashes), "duration", duration)
			filesByHash := map[string]torrent_stream.Files{}
			cached := map[string]bool{}
			for _, item := range res.Data.Items {
				res_item := store.CheckMagnetDataItem{
					Hash:   item.Hash,
					Magnet: item.Magnet,
					Status: item.Status,
				}
				res_files := []store.MagnetFile{}
				files := torrent_stream.Files{}
				if item.Status == store.MagnetStatusCached {
					cached[item.Hash] = true
					seenByName := map[string]struct{}{}
					for _, f := range item.Files {
						key := f.Path
						if key == "" {
							key = f.Name
						}
						if _, seen := seenByName[key]; seen {
							buddyLog.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
							continue
						}
						seenByName[key] = struct{}{}
						res_files = append(res_files, store.MagnetFile{
							Idx:    f.Idx,
							Path:   f.Path,
							Name:   f.Name,
							Size:   f.Size,
							Source: f.Source,
						})
						files = append(files, torrent_stream.File{
							Idx:    f.Idx,
							Path:   f.Path,
							Name:   f.Name,
							Size:   f.Size,
							SId:    f.SId,
							Source: f.Source,
						})
					}
				}
				res_item.Files = res_files
				data.Items = append(data.Items, res_item)
				filesByHash[item.Hash] = files
			}
			go magnet_cache.BulkTouch(s.GetName().Code(), filesByHash, cached, false)
			return data, nil
		}
	}

	if config.HasPeer {
		if config.PeerFlag.Lazy {
			storeCode := string(s.GetName().Code())
			for _, hash := range staleOrMissingHashes {
				worker_queue.MagnetCachePullerQueue.Queue(worker_queue.MagnetCachePullerQueueItem{
					ClientIP:   clientIp,
					Hash:       hash,
					SId:        sid,
					StoreCode:  storeCode,
					StoreToken: storeToken,
				})
			}
			return data, nil
		}

		if Peer.IsHaltedCheckMagnet() {
			return data, nil
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		filesByHash := map[string]torrent_stream.Files{}
		for cHashes := range slices.Chunk(staleOrMissingHashes, 500) {
			wg.Go(func() {

				params := &peer.CheckMagnetParams{
					StoreName:  s.GetName(),
					StoreToken: storeToken,
				}
				params.Magnets = cHashes
				params.ClientIP = clientIp
				params.SId = sid
				start := time.Now()
				res, err := Peer.CheckMagnet(params)
				duration := time.Since(start)
				if duration.Seconds() > 10 {
					Peer.HaltCheckMagnet()
				}
				if err != nil {
					peerLog.Error("failed partially to check magnet", "store.name", s.GetName(), "error", core.PackError(err), "duration", duration)
				} else {
					mu.Lock()
					defer mu.Unlock()

					peerLog.Info("check magnet", "store.name", s.GetName(), "hash_count", len(cHashes), "duration", duration)
					for _, item := range res.Data.Items {
						files := torrent_stream.Files{}
						if item.Status == store.MagnetStatusCached {
							seenByName := map[string]struct{}{}
							for _, f := range item.Files {
								key := f.Path
								if key == "" {
									key = f.Name
								}
								if _, seen := seenByName[key]; seen {
									peerLog.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
									continue
								}
								seenByName[key] = struct{}{}
								files = append(files, torrent_stream.File{
									Idx:       f.Idx,
									Path:      f.Path,
									Name:      f.Name,
									Size:      f.Size,
									Source:    f.Source,
									VideoHash: f.VideoHash,
									MediaInfo: f.MediaInfo,
								})
							}
						}
						filesByHash[item.Hash] = files
						data.Items = append(data.Items, item)
					}
				}
			})
		}
		wg.Wait()
		go magnet_cache.BulkTouch(s.GetName().Code(), filesByHash, nil, false)
		return data, nil
	}

	return data, nil
}
