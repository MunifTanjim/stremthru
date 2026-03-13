package worker

import (
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/magnet_cache"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/store"
)

func InitMagnetCachePullerWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		worker_queue.MagnetCachePullerQueue.ProcessGroup(func(key string, items []worker_queue.MagnetCachePullerQueueItem) error {
			storeCode, sid, _ := strings.Cut(key, ":")

			s := shared.GetStoreByCode(storeCode)
			if s == nil {
				w.Log.Error("invalid store code", "store.code", storeCode)
				return nil
			}

			hashes := make([]string, len(items))
			clientIps := []string{}
			seenClientIp := map[string]struct{}{}
			storeTokens := []string{}
			seenStoreToken := map[string]struct{}{}
			for i := range items {
				item := &items[i]
				hashes[i] = item.Hash
				if _, seen := seenClientIp[item.ClientIP]; !seen {
					clientIps = append(clientIps, item.ClientIP)
					seenClientIp[item.ClientIP] = struct{}{}
				}
				if _, seen := seenStoreToken[item.StoreToken]; !seen {
					storeTokens = append(storeTokens, item.StoreToken)
					seenStoreToken[item.StoreToken] = struct{}{}
				}
			}

			// Skip remote peer lookups for qBittorrent store (custom fork feature, not supported upstream)
			if s.GetName() == store.StoreNameQBittorrent {
				return nil
			}

			for i, cHashes := range slices.Collect(slices.Chunk(hashes, 500)) {
				if buddy.Peer.IsHaltedCheckMagnet() {
					time.Sleep(15 * time.Second)
				}

				filesByHash := map[string]torrent_stream.Files{}
				cached := map[string]bool{}

				storeToken := storeTokens[i%len(storeTokens)]
				clientIp := clientIps[i%len(clientIps)]

				params := &peer.CheckMagnetParams{
					StoreName:  s.GetName(),
					StoreToken: storeToken,
				}
				params.Magnets = cHashes
				params.ClientIP = clientIp
				params.SId = sid
				start := time.Now()
				res, err := buddy.Peer.CheckMagnet(params)
				duration := time.Since(start)
				if duration.Seconds() > 10 {
					Peer.HaltCheckMagnet()
				}
				if err != nil {
					w.Log.Error("failed partially to check magnet", "store.name", s.GetName(), "error", core.PackError(err), "duration", duration)
				} else {
					w.Log.Info("check magnet", "store.name", s.GetName(), "hash_count", len(cHashes), "duration", duration)
					for _, item := range res.Data.Items {
						files := torrent_stream.Files{}
						if item.Status == store.MagnetStatusCached {
							cached[item.Hash] = true
							seenByName := map[string]bool{}
							for _, f := range item.Files {
								if _, seen := seenByName[f.Name]; seen {
									w.Log.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
									continue
								}
								seenByName[f.Name] = true
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
					}
				}

				magnet_cache.BulkTouch(s.GetName().Code(), filesByHash, cached, false)
			}

			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
