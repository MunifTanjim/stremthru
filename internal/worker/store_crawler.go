package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/store"
)

func InitCrawlStoreWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		worker_queue.StoreCrawlerQueue.Process(func(item worker_queue.StoreCrawlerQueueItem) error {
			s := shared.GetStoreByCode(item.StoreCode)
			if s == nil {
				return nil
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

			return nil
		})

		return nil

	}

	worker := NewWorker(conf)

	return worker
}
