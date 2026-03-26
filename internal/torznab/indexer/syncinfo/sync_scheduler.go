package torznab_indexer_syncinfo

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
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

		interval := j.Interval()
		timeLimit := interval - interval/10
		startTime := time.Now()

		type indexerTask struct {
			indexer           *torznab_indexer.TorznabIndexer
			client            tznc.Indexer
			rateLimitExceeded atomic.Bool
		}

		processIndexerTask := func(task *indexerTask) (hasMore bool) {
			indexer := task.indexer
			client := task.client

			if task.rateLimitExceeded.Load() {
				log.Warn("rate limit exceeded, stopping indexer", "indexer", indexer.Name)
				return false
			}

			if time.Since(startTime) >= timeLimit {
				log.Info("time limit reached, stopping indexer", "indexer", indexer.Name)
				return false
			}

			if !torznab_indexer.IsEnabled(indexer.Id) {
				log.Info("indexer disabled, stopping", "indexer", indexer.Name)
				return false
			}

			rl, err := indexer.GetRateLimiter()
			if err != nil {
				log.Error("failed to get rate limiter", "error", err, "id", indexer.Id)
				return false
			}

			items, err := GetSyncPendingByIndexer(indexer.Id)
			if err != nil {
				log.Error("failed to get pending sync", "error", err, "indexer", indexer.Name)
				return false
			}

			if len(items) == 0 {
				log.Debug("no pending sync items", "indexer", indexer.Name)
				return false
			}

			log.Info("processing items for indexer", "indexer", indexer.Name, "count", len(items))

			var recordProcessMutex sync.Mutex
			var madeProgress atomic.Bool

			itemPool := pond.NewPool(3)
			for i := range items {
				if task.rateLimitExceeded.Load() {
					break
				}

				item := &items[i]
				itemPool.Submit(func() {
					if time.Since(startTime) >= timeLimit {
						log.Trace("time limit reached, skipping item", "indexer", indexer.Name, "sid", item.SId)
						return
					}

					queries := item.Queries
					if len(queries) == 0 {
						log.Debug("no queries stored for item", "sid", item.SId)
						return
					}

					recordProgress := func(queries Queries, query *Query) {
						recordProcessMutex.Lock()
						defer recordProcessMutex.Unlock()

						if err := RecordProgress(indexer.Id, item.SId, queries); err != nil {
							log.Error("failed to record progress", "error", err, "indexer", indexer.Name, "sid", item.SId, "query", query.Query)
						}
					}

					nsid, err := torrent_stream.NormalizeStreamId(item.SId)
					if err != nil {
						log.Error("failed to normalize stream id", "error", err, "sid", item.SId)
						queries[0].Error = fmt.Errorf("failed to normalize stream id: %w", err).Error()
						recordProgress(queries, &queries[0])
						return
					}

					results := []tznc.Torz{}

					for i := range queries {
						if task.rateLimitExceeded.Load() {
							break
						}

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
								if timeLeft := timeLimit - time.Since(startTime); result.RetryAfter > timeLeft {
									task.rateLimitExceeded.Store(true)
									log.Warn("rate limited, stopping indexer processing", "indexer", indexer.Name, "retry_after", result.RetryAfter.String())
									return
								}
								if err := rl.Wait(); err != nil {
									task.rateLimitExceeded.Store(true)
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
							if e, ok := errors.AsType[*znab.Error](err); ok {
								switch e.StatusCode {
								case http.StatusBadRequest:
									if strings.Contains(e.Description, "429 Too Many Requests") {
										task.rateLimitExceeded.Store(true)
										log.Warn("too many requests detected from bad request error, stopping indexer processing", "indexer", indexer.Name)
										return
									}
								case http.StatusTooManyRequests:
									if e.RetryAfter > 0 {
										if timeLeft := timeLimit - time.Since(startTime); e.RetryAfter < timeLeft {
											log.Warn("too many requests, waiting before retrying", "indexer", indexer.Name, "retry_after", e.RetryAfter.String())
											time.Sleep(e.RetryAfter)
											continue
										}
									}
									task.rateLimitExceeded.Store(true)
									log.Warn("too many requests, stopping indexer processing", "indexer", indexer.Name)
									return
								}
							}
							log.Error("indexer search failed", "error", err, "indexer", indexer.Name, "query", sQuery.Query, "duration", time.Since(start).String())
							sQuery.Error = err.Error()
							recordProgress(queries, sQuery)
							continue
						}

						sQuery.Count = len(qResults)
						sQuery.Done = true
						sQuery.Error = ""
						madeProgress.Store(true)

						log.Debug("indexer search completed", "indexer", indexer.Name, "query", sQuery.Query, "duration", time.Since(start).String(), "count", sQuery.Count)

						recordProgress(queries, sQuery)

						results = append(results, qResults...)
					}

					if task.rateLimitExceeded.Load() && len(results) == 0 {
						log.Trace("indexer search rate limited", "indexer", indexer.Name, "sid", item.SId, "count", len(results))
					} else {
						log.Debug("indexer search completed", "indexer", indexer.Name, "sid", item.SId, "count", len(results))
					}

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
							return
						}

						log.Debug("saved torrents", "indexer", indexer.Name, "sid", item.SId, "count", len(tInfosToUpsert))
					}
				})
			}
			if err := itemPool.Stop().Wait(); err != nil {
				log.Error("errors during item processing", "error", err, "indexer", indexer.Name)
			}

			return madeProgress.Load()
		}

		indexers, err := torznab_indexer.GetAllEnabled()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		if len(indexers) == 0 {
			log.Debug("no enabled indexers")
			return nil
		}

		var tasks []*indexerTask
		for i := range indexers {
			indexer := &indexers[i]

			var client tznc.Indexer
			switch indexer.Type {
			case torznab_indexer.IndexerTypeGeneric, torznab_indexer.IndexerTypeJackett:
				c, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "id", indexer.Id)
					continue
				}
				client = c
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
				continue
			}

			tasks = append(tasks, &indexerTask{
				indexer: indexer,
				client:  client,
			})
		}

		indexerTaskPool := pond.NewPool(5)
		var taskWg sync.WaitGroup

		var submitTask func(task *indexerTask)
		submitTask = func(task *indexerTask) {
			indexerTaskPool.Submit(func() {
				defer func() {
					if perr, stack := util.HandlePanic(recover(), true); perr != nil {
						log.Error("indexer task panic", "error", perr, "stack", stack, "indexer", task.indexer.Name)
						taskWg.Done()
					}
				}()
				hasMore := processIndexerTask(task)
				if hasMore {
					submitTask(task)
				} else {
					taskWg.Done()
				}
			})
		}

		for _, task := range tasks {
			taskWg.Add(1)
			submitTask(task)
		}

		taskWg.Wait()
		indexerTaskPool.Stop().Wait()

		return nil
	},
})
