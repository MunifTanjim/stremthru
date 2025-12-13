package worker

import (
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

func InitTorznabIndexerSyncerQueueWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		worker_queue.TorznabIndexerSyncerQueue.Process(func(item worker_queue.TorznabIndexerSyncerQueueItem) error {
			indexers, err := torznab_indexer.GetAll()
			if err != nil {
				log.Error("failed to get indexers", "error", err)
				return err
			}

			if len(indexers) == 0 {
				log.Debug("no indexers configured")
				return nil
			}

			for i := range indexers {
				indexer := &indexers[i]
				torznab_indexer_syncinfo.Queue(indexer.Type, indexer.Id, item.SId)
			}

			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
