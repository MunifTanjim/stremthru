package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/madflojo/tasks"
)

type StoreCrawlerQueueItem struct {
	StoreCode  string
	StoreToken string
}

var StoreCrawlerQueue = WorkerQueue[StoreCrawlerQueueItem]{
	debounceTime: 15 * time.Minute,
	getKey: func(item StoreCrawlerQueueItem) string {
		return item.StoreCode + ":" + item.StoreToken
	},
	transform: func(item *StoreCrawlerQueueItem) *StoreCrawlerQueueItem {
		return item
	},
}

func InitCrawlStoreWorker() *tasks.Scheduler {
	log := logger.Scoped("worker/store_crawler")

	scheduler := tasks.New()

	id, err := scheduler.Add(&tasks.Task{
		Interval:          time.Duration(30 * time.Minute),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.RecoverPanic(true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				}
			}()

			StoreCrawlerQueue.process(func(item StoreCrawlerQueueItem) {
				s := shared.GetStoreByCode(item.StoreCode)
				if s == nil {
					return
				}

				tSource := torrent_info.TorrentInfoSource(item.StoreCode)
				discardFileIdx := s.GetName().Code() != store.StoreCodeRealDebrid

				limit := 500
				offset := 0
				totalItems := 0
				for {
					params := &store.ListMagnetsParams{
						Limit:  limit,
						Offset: offset,
					}
					params.APIKey = item.StoreToken
					res, err := s.ListMagnets(params)
					if err != nil {
						log.Error("failed to list magnets", "err", err)
						break
					}

					if len(res.Items) == 0 {
						break
					}

					tInfos := []torrent_info.TorrentInfoInsertData{}
					for i := range res.Items {
						item := &res.Items[i]
						tInfo := torrent_info.TorrentInfoInsertData{
							Hash:         item.Hash,
							TorrentTitle: item.Name,
							Size:         item.Size,
							Source:       tSource,
						}
						tInfos = append(tInfos, tInfo)
					}
					torrent_info.Upsert(tInfos, "", discardFileIdx)

					totalItems += len(res.Items)
					if res.TotalItems <= totalItems {
						break
					}

					offset += limit

					time.Sleep(2 * time.Second)
				}
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

	return scheduler
}
