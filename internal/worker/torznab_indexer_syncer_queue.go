package worker

import (
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	znabsearch "github.com/MunifTanjim/stremthru/internal/znab/search"
)

func InitTorznabIndexerSyncerQueueWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		indexers, err := torznab_indexer.GetAllEnabled()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		clientById := map[int64]tznc.Indexer{}
		for i := range indexers {
			indexer := &indexers[i]

			switch indexer.Type {
			case torznab_indexer.IndexerTypeJackett:
				client, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "id", indexer.Id)
					continue
				}
				clientById[indexer.Id] = client
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
			}
		}

		worker_queue.TorznabIndexerSyncerQueue.Process(func(item worker_queue.TorznabIndexerSyncerQueueItem) error {
			meta, nsid, err := znabsearch.GetQueryMeta(log, item.SId)
			if err != nil {
				log.Error("failed to get query metadata", "error", err, "sid", item.SId)
				return nil
			}
			if len(meta.Titles) == 0 {
				log.Debug("no titles found for stream", "sid", item.SId)
				return nil
			}

			for i := range indexers {
				indexer := &indexers[i]

				client, ok := clientById[indexer.Id]
				if !ok {
					continue
				}

				queriesBySid, err := znabsearch.BuildQueriesForTorznab(client, znabsearch.QueryBuilderConfig{
					Meta: meta,
					NSId: nsid,
				})
				if err != nil {
					log.Error("failed to build queries for indexer", "error", err, "indexer", indexer.Name, "sid", item.SId)
					continue
				}

				if len(queriesBySid) == 0 {
					log.Debug("no queries generated for indexer", "indexer", indexer.Name, "sid", item.SId)
					continue
				}

				totalQueued := 0
				for sid, queries := range queriesBySid {
					queryItems := make(torznab_indexer_syncinfo.Queries, len(queries))
					for i := range queries {
						queryItems[i] = torznab_indexer_syncinfo.Query{
							Query: queries[i].Query.Encode(),
							Exact: queries[i].IsExact,
						}
					}

					count := len(queries)
					err = torznab_indexer_syncinfo.Queue(indexer.Id, sid, queryItems)
					if err != nil {
						log.Error("failed to queue sync", "error", err, "indexer", indexer.Name, "sid", sid, "query_count", count)
						continue
					}
					totalQueued += count
					log.Debug("queued sync", "indexer", indexer.Name, "sid", sid, "query_count", count)
				}
				log.Info("queued torznab indexer sync", "indexer", indexer.Name, "sid", item.SId, "query_count", totalQueued)
			}

			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
