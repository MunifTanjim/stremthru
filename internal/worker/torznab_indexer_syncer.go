package worker

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
)

func InitTorznabIndexerSyncerWorker(conf *WorkerConfig) *Worker {
	type indexerQueryMeta struct {
		titles     []string
		year       int
		season, ep int
	}
	type indexerQuery struct {
		query    *tznc.Query
		is_exact bool
	}
	conf.Executor = func(w *Worker) error {
		log := w.Log

		pendingItems, err := torznab_indexer_syncinfo.GetSyncPending()
		if err != nil {
			log.Error("failed to get pending sync", "error", err)
			return err
		}

		if len(pendingItems) == 0 {
			log.Debug("no pending sync items")
			return nil
		}

		indexers, err := torznab_indexer.GetAll()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		log.Info("processing pending sync items", "count", len(pendingItems))

		itemsByIndexer := make(map[string][]torznab_indexer_syncinfo.TorznabIndexerSyncInfo)
		for _, item := range pendingItems {
			key := string(item.Type) + ":" + item.Id
			itemsByIndexer[key] = append(itemsByIndexer[key], item)
		}

		indexerById := make(map[string]*torznab_indexer.TorznabIndexer)
		for i := range indexers {
			indexer := &indexers[i]
			key := string(indexer.Type) + ":" + indexer.Id
			indexerById[key] = indexer
		}

		var wg sync.WaitGroup
		for key, items := range itemsByIndexer {
			indexer, ok := indexerById[key]
			if !ok {
				log.Warn("indexer not found in vault", "key", key)
				continue
			}

			var client tznc.Indexer
			switch indexer.Type {
			case torznab_indexer.IndexerTypeJackett:
				c, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "type", indexer.Type, "id", indexer.Id)
					return err
				}
				client = c
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
				continue
			}

			wg.Go(func() {
				log.Info("processing items for indexer", "indexer", indexer.Name, "count", len(items))
				for i := range items {
					item := &items[i]

					queryMeta := indexerQueryMeta{
						titles: []string{},
					}

					nsid, err := torrent_stream.NormalizeStreamId(item.SId)
					if err != nil {
						log.Error("failed to normalize stream ID", "error", err, "sid", item.SId)
						continue
					}
					if nsid.IsAnime {
						if aniEp := util.SafeParseInt(nsid.Episode, -1); aniEp != -1 {
							tvdbMaps, err := anidb.GetTVDBEpisodeMaps(nsid.Id, false)
							if err != nil {
								log.Error("failed to get AniDB-TVDB episode maps", "error", err, "anidb_id", nsid.Id)
								continue
							}
							if epMap := tvdbMaps.GetByAnidbEpisode(aniEp); epMap != nil {
								ep := epMap.GetTMDBEpisode(aniEp)
								titles, err := anidb.GetTitlesByIds([]string{nsid.Id})
								if err != nil {
									log.Error("failed to get AniDB titles", "error", err, "anidb_id", nsid.Id)
									continue
								}
								if len(titles) == 0 {
									log.Warn("AniDB title not found", "anidb_id", nsid.Id)
									continue
								}
								queryMeta.titles = make([]string, 0, len(titles))
								queryMeta.season = epMap.TVDBSeason
								queryMeta.ep = ep
								seenTitle := util.NewSet[string]()
								for i := range titles {
									title := &titles[i]
									if seenTitle.Has(title.Value) {
										continue
									}
									seenTitle.Add(title.Value)
									queryMeta.titles = append(queryMeta.titles, title.Value)
									if queryMeta.year == 0 && title.Year != "" {
										queryMeta.year = util.SafeParseInt(title.Year, 0)
									}
								}
							}
						}
					} else {
						it, err := imdb_title.Get(nsid.Id)
						if err != nil {
							log.Error("failed to get IMDB title", "error", err, "imdb_id", nsid.Id)
							continue
						}
						if it == nil {
							log.Warn("IMDB title not found", "imdb_id", nsid.Id)
							continue
						}
						queryMeta.titles = append(queryMeta.titles, it.Title)
						if it.OrigTitle != "" && it.OrigTitle != it.Title {
							queryMeta.titles = append(queryMeta.titles, it.OrigTitle)
						}
						if it.Year > 0 {
							queryMeta.year = it.Year
						}
						if nsid.IsSeries() {
							queryMeta.season = util.SafeParseInt(nsid.Season, 0)
							queryMeta.ep = util.SafeParseInt(nsid.Episode, 0)
						}
					}

					sQueriesBySId := map[string][]indexerQuery{}
					query, err := client.NewSearchQuery(func(caps tznc.Caps) tznc.Function {
						if nsid.IsSeries() && caps.SupportsFunction(tznc.FunctionSearchTV) {
							return tznc.FunctionSearchTV
						}
						if caps.SupportsFunction(tznc.FunctionSearchMovie) {
							return tznc.FunctionSearchMovie
						}
						return tznc.FunctionSearch
					})
					if err != nil {
						log.Error("failed to create search query", "error", err, "indexer", client.GetId())
						continue
					}

					query.SetLimit(-1)
					if !nsid.IsAnime && query.IsSupported(tznc.SearchParamIMDBId) {
						query.Set(tznc.SearchParamIMDBId, nsid.Id)
						sid := nsid.ToClean()
						sQuery := indexerQuery{
							query:    query,
							is_exact: !nsid.IsSeries(),
						}
						if nsid.IsSeries() {
							if query.IsSupported(tznc.SearchParamSeason) && nsid.Season != "" {
								query.Set(tznc.SearchParamSeason, nsid.Season)
								if query.IsSupported(tznc.SearchParamEp) && nsid.Episode != "" {
									query.Set(tznc.SearchParamEp, nsid.Episode)
									sQuery.is_exact = true
									sid = nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
								} else {
									sid = nsid.ToClean() + ":" + nsid.Season
								}
							}
						}
						sQueriesBySId[sid] = append(sQueriesBySId[sid], sQuery)
					} else {
						query.SetT(tznc.FunctionSearch)
						supportsYear := query.IsSupported(tznc.SearchParamYear)
						if supportsYear && queryMeta.year != 0 {
							query.Set(tznc.SearchParamYear, strconv.Itoa(queryMeta.year))
						}
						for _, title := range queryMeta.titles {
							var q strings.Builder
							q.WriteString(title)
							if nsid.IsSeries() {
								sid := nsid.ToClean()
								sQueriesBySId[sid] = append(sQueriesBySId[sid], indexerQuery{
									query: query.Clone().Set(tznc.SearchParamQ, q.String()),
								})
								if queryMeta.season > 0 {
									q.WriteString(" S")
									q.WriteString(util.ZeroPadInt(queryMeta.season, 2))
									sid := nsid.ToClean() + ":" + nsid.Season
									sQueriesBySId[sid] = append(sQueriesBySId[sid], indexerQuery{
										query: query.Clone().Set(tznc.SearchParamQ, q.String()),
									})
									if queryMeta.ep > 0 {
										q.WriteString("E")
										q.WriteString(util.ZeroPadInt(queryMeta.ep, 2))
										sid := nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
										sQueriesBySId[sid] = append(sQueriesBySId[sid], indexerQuery{
											query: query.Clone().Set(tznc.SearchParamQ, q.String()),
										})
									}
								}
							} else if queryMeta.year > 0 {
								if !supportsYear {
									q.WriteString(" ")
									q.WriteString(strconv.Itoa(queryMeta.year))
								}
								sid := nsid.ToClean()
								sQueriesBySId[sid] = append(sQueriesBySId[sid], indexerQuery{
									query: query.Clone().Set(tznc.SearchParamQ, q.String()),
								})
							}
						}
					}

					results := []tznc.Torz{}

					for sid, sQueries := range sQueriesBySId {
						if !torznab_indexer_syncinfo.ShouldSync(item.Type, item.Id, sid) {
							log.Debug("skipping already synced query", "indexer", client.GetId(), "sid", sid)
							continue
						}

						var wg sync.WaitGroup
						cResults := make([][]tznc.Torz, len(sQueries))
						errs := make([]error, len(sQueries))
						for i := range sQueries {
							sQuery := sQueries[i]
							wg.Go(func() {
								start := time.Now()
								cResults[i], errs[i] = client.Search(sQuery.query)
								if errs[i] == nil {
									log.Debug("indexer search completed", "indexer", client.GetId(), "query", sQuery.query.Encode(), "duration", time.Since(start).String(), "count", len(cResults[i]))
								} else {
									log.Error("indexer search failed", "error", errs[i], "indexer", client.GetId(), "query", sQuery.query.Encode(), "duration", time.Since(start).String())
								}
							})
						}
						wg.Wait()

						if err := errors.Join(errs...); err != nil {
							log.Error("some indexer search failed", "indexer", client.GetId(), "sid", item.SId, "error", err)
							if err := torznab_indexer_syncinfo.SetSyncError(item.Type, item.Id, sid, err.Error()); err != nil {
								log.Error("failed to set sync error", "error", err, "type", item.Type, "id", item.Id, "sid", sid)
							}
							continue
						}

						resultCount := 0
						for _, items := range cResults {
							resultCount += len(items)
							results = append(results, items...)
						}

						if err := torznab_indexer_syncinfo.MarkSynced(item.Type, item.Id, sid, resultCount); err != nil {
							log.Error("failed to mark synced", "error", err, "type", item.Type, "id", item.Id, "sid", sid)
						}
					}

					log.Debug("indexer search completed", "indexer", client.GetId(), "sid", item.SId, "count", len(results))

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

						log.Debug("saved torrents", "indexer", client.GetId(), "sid", item.SId, "count", len(tInfosToUpsert))
					}
				}
			})
		}
		wg.Wait()

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
