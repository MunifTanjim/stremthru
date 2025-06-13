package worker

import (
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/magnet_cache"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/madflojo/tasks"
)

func InitMagnetCachePullerWorker(conf *WorkerConfig) *Worker {
	if worker_queue.MagnetCachePullerQueue.Disabled {
		return nil
	}

	log := logger.Scoped("worker/magnet_cache_puller")

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(5 * time.Minute),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				}
				worker.onEnd()
			}()

			for {
				wait, reason := worker.shouldWait()
				if !wait {
					break
				}
				log.Info("waiting, " + reason)
				time.Sleep(5 * time.Minute)
			}
			worker.onStart()

			worker_queue.MagnetCachePullerQueue.ProcessGroup(func(key string, items []worker_queue.MagnetCachePullerQueueItem) error {
				storeCode, sid, _ := strings.Cut(key, ":")

				s := shared.GetStoreByCode(storeCode)
				if s == nil {
					log.Error("invalid store code", "store_code", storeCode)
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

				for i, cHashes := range slices.Collect(slices.Chunk(hashes, 500)) {
					if buddy.Peer.IsHaltedCheckMagnet() {
						time.Sleep(15 * time.Second)
					}

					filesByHash := map[string]magnet_cache.Files{}

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
						log.Error("failed partially to check magnet", "store", s.GetName(), "error", core.PackError(err), "duration", duration)
					} else {
						log.Info("check magnet", "store", s.GetName(), "hash_count", len(cHashes), "duration", duration)
						for _, item := range res.Data.Items {
							files := magnet_cache.Files{}
							if item.Status == store.MagnetStatusCached {
								seenByName := map[string]bool{}
								for _, f := range item.Files {
									if _, seen := seenByName[f.Name]; seen {
										log.Info("found duplicate file", "hash", item.Hash, "filename", f.Name)
										continue
									}
									seenByName[f.Name] = true
									files = append(files, magnet_cache.File{Idx: f.Idx, Name: f.Name, Size: f.Size})
								}
							}
							filesByHash[item.Hash] = files
						}
					}

					magnet_cache.BulkTouch(s.GetName().Code(), filesByHash, false)
				}

				return nil
			})

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	return worker
}
