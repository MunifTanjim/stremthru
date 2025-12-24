package worker

import (
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

func InitTorznabIndexerSyncerQueueWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		indexers, err := torznab_indexer.GetAll()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		worker_queue.TorznabIndexerSyncerQueue.Process(func(item worker_queue.TorznabIndexerSyncerQueueItem) error {
			for i := range indexers {
				indexer := &indexers[i]
				err := torznab_indexer_syncinfo.Queue(indexer.Type, indexer.Id, item.SId)
				if err != nil {
					return err
				}
			}
			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
