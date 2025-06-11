package worker

import (
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

func InitMapAniDBTorrentWorker(conf *WorkerConfig) *Worker {
	if !config.Feature.IsEnabled("anime") {
		return nil
	}

	log := logger.Scoped("worker/map_anidb_torrent")

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	isRunning := false
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(1 * time.Hour),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else {
					isRunning = false
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

			if isRunning {
				return nil
			}

			isRunning = true

			batch_size := 10000
			chunk_size := 1000
			if db.Dialect == db.DBDialectPostgres {
				batch_size = 20000
				chunk_size = 2000
			}

			totalCount := 0
			for {
				hashes, err := torrent_info.GetAniDBUnmappedHashes(batch_size)
				if err != nil {
					return err
				}

				var wg sync.WaitGroup
				for cHashes := range slices.Chunk(hashes, chunk_size) {
					wg.Add(1)
					go func() {
						defer wg.Done()

						items := []anidb.AniDBTorrent{}
						tInfoByHash, err := torrent_info.GetByHashes(cHashes)
						if err != nil {
							log.Error("failed to get torrent info", "error", err)
							return
						}
						for hash, tInfo := range tInfoByHash {
							if !tInfo.IsParsed() {
								continue
							}

							if tInfo.Title == "" {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								continue
							}

							anidbTitleIds, err := anidb.SearchIdsByTitle(tInfo.Title, tInfo.Seasons, 0)
							if err != nil {
								log.Error("failed to search anidb title", "error", err, "title", tInfo.Title)
								continue
							}
							if len(anidbTitleIds) == 0 {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
							} else {
								for _, tid := range anidbTitleIds {
									items = append(items, anidb.AniDBTorrent{
										TId:  tid,
										Hash: hash,
									})
								}
							}
						}

						if err := anidb.InsertTorrents(items); err != nil {
							log.Error("failed to map anidb torrent", "error", err)
							return
						}

						log.Info("mapped anidb torrent", "count", len(items))
					}()
				}
				wg.Wait()

				count := len(hashes)
				totalCount += count
				log.Info("processed torrents", "totalCount", totalCount)

				if count < batch_size {
					break
				}

				time.Sleep(200 * time.Millisecond)
			}

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			isRunning = false
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	if task, err := worker.scheduler.Lookup(id); err == nil && task != nil {
		t := task.Clone()
		t.Interval = 60 * time.Second
		t.RunOnce = true
		worker.scheduler.Add(t)
	}

	return worker
}
