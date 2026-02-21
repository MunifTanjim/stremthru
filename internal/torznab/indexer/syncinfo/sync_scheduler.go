package torznab_indexer_syncinfo

import (
	"net/url"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
)

const syncSchedulerId = "sync-torznab-indexer"

var _ = job.NewScheduler(&job.SchedulerConfig[JobData]{
	Id:           syncSchedulerId,
	Title:        "Sync Torznab Indexer",
	Interval:     30 * time.Minute,
	RunExclusive: true,
	Disabled:     !config.Feature.HasVault(),
	ShouldSkip: func() bool {
		return !HasSyncPending()
	},
	Executor: func(j *job.Scheduler[JobData]) error {
		log := j.Logger()

		rateLimitTotalWaitThreshold := 15 * time.Minute

		pendingItems, err := GetSyncPending()
		if err != nil {
			log.Error("failed to get pending sync", "error", err)
			return err
		}

		if len(pendingItems) == 0 {
			log.Debug("no pending sync items")
			return nil
		}

		indexers, err := torznab_indexer.GetAllEnabled()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		log.Info("processing pending sync items", "count", len(pendingItems))

		itemsByIndexerId := make(map[int64][]TorznabIndexerSyncInfo)
		for _, item := range pendingItems {
			itemsByIndexerId[item.IndexerId] = append(itemsByIndexerId[item.IndexerId], item)
		}

		indexerById := make(map[int64]*torznab_indexer.TorznabIndexer)
		for i := range indexers {
			indexer := &indexers[i]
			indexerById[indexer.Id] = indexer
		}

		var wg sync.WaitGroup
		for indexerId, items := range itemsByIndexerId {
			indexer, ok := indexerById[indexerId]
			if !ok {
				log.Warn("indexer not found in vault", "id", indexerId)
				continue
			}

			var client tznc.Indexer
			switch indexer.Type {
			case torznab_indexer.IndexerTypeJackett:
				c, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "id", indexer.Id)
					return err
				}
				client = c
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
				continue
			}

			wg.Go(func() {
				log.Info("processing items for indexer", "indexer", indexer.Name, "count", len(items))

				rl, err := indexer.GetRateLimiter()
				if err != nil {
					log.Error("failed to get rate limiter", "error", err, "id", indexer.Id)
					return
				}

				rateLimitedWait := 0 * time.Second

				for i := range items {
					item := &items[i]

					queries := item.Queries
					if len(queries) == 0 {
						log.Debug("no queries stored for item", "sid", item.SId)
						continue
					}

					nsid, err := torrent_stream.NormalizeStreamId(item.SId)
					if err != nil {
						log.Error("failed to normalize stream id", "error", err, "sid", item.SId)
						continue
					}

					results := []tznc.Torz{}

					recordProgress := func(queries Queries, query *Query) {
						if err := RecordProgress(indexer.Id, item.SId, queries); err != nil {
							log.Error("failed to record progress", "error", err, "indexer", indexer.Name, "sid", item.SId, "query", query.Query)
						}
					}

					for i := range queries {
						sQuery := &queries[i]
						if sQuery.Done {
							continue
						}

						query, err := url.ParseQuery(sQuery.Query)
						if err != nil {
							log.Error("failed to parse query", "error", err, "indexer", indexer.Name, "query", sQuery.Query)
							sQuery.Error = err.Error()
							recordProgress(queries, sQuery)
							continue
						}

						if rl != nil {
							if result, err := rl.Try(); err != nil {
								log.Error("rate limit check failed", "error", err, "indexer", indexer.Name)
								sQuery.Error = err.Error()
								recordProgress(queries, sQuery)
								continue
							} else if !result.Allowed {
								if rateLimitedWait+result.RetryAfter > rateLimitTotalWaitThreshold {
									log.Warn("rate limited, stopping indexer processing", "indexer", indexer.Name, "retry_after", result.RetryAfter.String())
									return
								}
								rateLimitedWait += result.RetryAfter
								if err := rl.Wait(); err != nil {
									log.Error("rate limit wait failed", "error", err, "indexer", indexer.Name)
									sQuery.Error = err.Error()
									recordProgress(queries, sQuery)
									return
								}
							}
						}

						start := time.Now()
						qResults, err := client.Search(query)
						if err != nil {
							log.Error("indexer search failed", "error", err, "indexer", indexer.Name, "query", sQuery.Query, "duration", time.Since(start).String())
							sQuery.Error = err.Error()
							recordProgress(queries, sQuery)
							continue
						}

						sQuery.Count = len(qResults)
						sQuery.Done = true
						sQuery.Error = ""

						log.Debug("indexer search completed", "indexer", indexer.Name, "query", sQuery.Query, "duration", time.Since(start).String(), "count", sQuery.Count)

						recordProgress(queries, sQuery)

						results = append(results, qResults...)
					}

					log.Debug("indexer search completed", "indexer", indexer.Name, "sid", item.SId, "count", len(results))

					// TODO: download torrent files in a separate queue
					seenSourceURL := util.NewSet[string]()
					torzFetchWg := pond.NewPool(5)
					for i := range results {
						item := &results[i]
						if item.HasMissingData() && item.SourceLink != "" {
							if seenSourceURL.Has(item.SourceLink) {
								continue
							}
							seenSourceURL.Add(item.SourceLink)

							torzFetchWg.Submit(func() {
								err := item.EnsureMagnet()
								if err != nil {
									log.Warn("failed to ensure magnet link for torrent", "error", err)
								}
							})
						}
					}
					if err := torzFetchWg.Stop().Wait(); err != nil {
						log.Warn("errors occurred while fetching torrent magnets", "error", err)
					}

					tInfosToUpsert := []torrent_info.TorrentItem{}
					for i := range results {
						item := &results[i]
						if item.HasMissingData() {
							continue
						}

						tInfo := torrent_info.TorrentItem{
							Hash:         item.Hash,
							TorrentTitle: item.Title,
							Size:         item.Size,
							Indexer:      item.Indexer,
							Source:       torrent_info.TorrentInfoSourceIndexer,
							Seeders:      item.Seeders,
							Leechers:     item.Leechers,
							Private:      item.Private,
							Files:        item.Files,
						}
						tInfosToUpsert = append(tInfosToUpsert, tInfo)
					}

					if len(tInfosToUpsert) > 0 {
						category := torrent_info.TorrentInfoCategoryUnknown
						if nsid.IsSeries() {
							category = torrent_info.TorrentInfoCategorySeries
						} else {
							category = torrent_info.TorrentInfoCategoryMovie
						}

						if err := torrent_info.Upsert(tInfosToUpsert, category, false); err != nil {
							log.Error("failed to upsert torrent info", "error", err, "count", len(tInfosToUpsert))
							continue
						}

						log.Debug("saved torrents", "indexer", indexer.Name, "sid", item.SId, "count", len(tInfosToUpsert))
					}
				}
			})
		}
		wg.Wait()

		return nil
	},
})
