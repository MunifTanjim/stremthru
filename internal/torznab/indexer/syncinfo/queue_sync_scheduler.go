package torznab_indexer_syncinfo

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	znabsearch "github.com/MunifTanjim/stremthru/internal/znab/search"
)

const queueSyncSchedulerId = "queue-torznab-indexer-sync"

var _ = job.NewScheduler(&job.SchedulerConfig[JobData]{
	Id:           queueSyncSchedulerId,
	Title:        "Queue Torznab Indexer Sync",
	Interval:     10 * time.Minute,
	RunExclusive: true,
	Disabled:     !config.Feature.HasVault(),
	Queue:        queue,
	ShouldSkip: func() bool {
		return queue.IsEmpty() || !torznab_indexer.Exists()
	},
	Executor: func(j *job.Scheduler[JobData]) error {
		log := j.Logger()

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

		j.JobQueue().Process(func(item JobData) error {
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
					queryItems := make(Queries, len(queries))
					for i := range queries {
						queryItems[i] = Query{
							Query: queries[i].Query.Encode(),
							Exact: queries[i].IsExact,
						}
					}

					count := len(queries)
					err = Queue(indexer.Id, sid, queryItems)
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
	},
})
