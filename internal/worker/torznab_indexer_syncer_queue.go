package worker

import (
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

func InitTorznabIndexerSyncerQueueWorker(conf *WorkerConfig) *Worker {
	type queryMeta struct {
		titles     []string
		year       int
		season, ep int
	}

	getQueryMeta := func(log *logger.Logger, sid string) (*queryMeta, *torrent_stream.NormalizedStremId, error) {
		nsid, err := torrent_stream.NormalizeStreamId(sid)
		if err != nil {
			return nil, nil, err
		}

		meta := &queryMeta{
			titles: []string{},
		}

		if nsid.IsAnime {
			if aniEp := util.SafeParseInt(nsid.Episode, -1); aniEp != -1 {
				tvdbMaps, err := anidb.GetTVDBEpisodeMaps(nsid.Id, false)
				if err != nil {
					log.Error("failed to get AniDB-TVDB episode maps", "error", err, "anidb_id", nsid.Id)
					return nil, nsid, err
				}
				if epMap := tvdbMaps.GetByAnidbEpisode(aniEp); epMap != nil {
					ep := epMap.GetTMDBEpisode(aniEp)
					titles, err := anidb.GetTitlesByIds([]string{nsid.Id})
					if err != nil {
						log.Error("failed to get AniDB titles", "error", err, "anidb_id", nsid.Id)
						return nil, nsid, err
					}
					if len(titles) == 0 {
						log.Warn("AniDB title not found", "anidb_id", nsid.Id)
						return meta, nsid, nil
					}
					meta.titles = make([]string, 0, len(titles))
					meta.season = epMap.TVDBSeason
					meta.ep = ep
					seenTitle := util.NewSet[string]()
					for i := range titles {
						title := &titles[i]
						if seenTitle.Has(title.Value) {
							continue
						}
						seenTitle.Add(title.Value)
						meta.titles = append(meta.titles, title.Value)
						if meta.year == 0 && title.Year != "" {
							meta.year = util.SafeParseInt(title.Year, 0)
						}
					}
				}
			}
		} else {
			it, err := imdb_title.Get(nsid.Id)
			if err != nil {
				log.Error("failed to get IMDB title", "error", err, "imdb_id", nsid.Id)
				return nil, nsid, err
			}
			if it == nil {
				log.Warn("IMDB title not found", "imdb_id", nsid.Id)
				return meta, nsid, nil
			}
			meta.titles = append(meta.titles, it.Title)
			if it.OrigTitle != "" && it.OrigTitle != it.Title {
				meta.titles = append(meta.titles, it.OrigTitle)
			}
			if it.Year > 0 {
				meta.year = it.Year
			}
			if nsid.IsSeries() {
				meta.season = util.SafeParseInt(nsid.Season, 0)
				meta.ep = util.SafeParseInt(nsid.Episode, 0)
			}
		}

		return meta, nsid, nil
	}

	buildQueriesForIndexer := func(client tznc.Indexer, nsid *torrent_stream.NormalizedStremId, meta *queryMeta) (map[string]torznab_indexer_syncinfo.Queries, error) {
		queriesBySid := map[string]torznab_indexer_syncinfo.Queries{}

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
			return nil, err
		}

		query.SetLimit(-1)

		if !nsid.IsAnime && query.IsSupported(tznc.SearchParamIMDBId) {
			query.Set(tznc.SearchParamIMDBId, nsid.Id)
			sid := nsid.ToClean()
			isExact := !nsid.IsSeries()

			if nsid.IsSeries() {
				if query.IsSupported(tznc.SearchParamSeason) && nsid.Season != "" {
					query.Set(tznc.SearchParamSeason, nsid.Season)
					if query.IsSupported(tznc.SearchParamEp) && nsid.Episode != "" {
						query.Set(tznc.SearchParamEp, nsid.Episode)
						isExact = true
						sid = nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
					} else {
						sid = nsid.ToClean() + ":" + nsid.Season
					}
				}
			}

			queriesBySid[sid] = append(queriesBySid[sid], torznab_indexer_syncinfo.Query{
				Query: query.Encode(),
				Exact: isExact,
			})
		} else {
			query.SetT(tznc.FunctionSearch)
			supportsYear := query.IsSupported(tznc.SearchParamYear)
			if supportsYear && meta.year != 0 {
				query.Set(tznc.SearchParamYear, strconv.Itoa(meta.year))
			}

			for _, title := range meta.titles {
				var q strings.Builder
				q.WriteString(title)

				if nsid.IsSeries() {
					sid := nsid.ToClean()
					queriesBySid[sid] = append(queriesBySid[sid], torznab_indexer_syncinfo.Query{
						Query: query.Clone().Set(tznc.SearchParamQ, q.String()).Encode(),
					})

					if meta.season > 0 {
						q.WriteString(" S")
						q.WriteString(util.ZeroPadInt(meta.season, 2))
						sid := nsid.ToClean() + ":" + nsid.Season
						queriesBySid[sid] = append(queriesBySid[sid], torznab_indexer_syncinfo.Query{
							Query: query.Clone().Set(tznc.SearchParamQ, q.String()).Encode(),
						})

						if meta.ep > 0 {
							q.WriteString("E")
							q.WriteString(util.ZeroPadInt(meta.ep, 2))
							sid := nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
							queriesBySid[sid] = append(queriesBySid[sid], torznab_indexer_syncinfo.Query{
								Query: query.Clone().Set(tznc.SearchParamQ, q.String()).Encode(),
							})
						}
					}
				} else if meta.year > 0 {
					if !supportsYear {
						q.WriteString(" ")
						q.WriteString(strconv.Itoa(meta.year))
					}
					sid := nsid.ToClean()
					queriesBySid[sid] = append(queriesBySid[sid], torznab_indexer_syncinfo.Query{
						Query: query.Clone().Set(tznc.SearchParamQ, q.String()).Encode(),
					})
				}
			}
		}

		return queriesBySid, nil
	}

	conf.Executor = func(w *Worker) error {
		log := w.Log

		indexers, err := torznab_indexer.GetAll()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		clientById := map[string]tznc.Indexer{}
		for i := range indexers {
			indexer := &indexers[i]

			switch indexer.Type {
			case torznab_indexer.IndexerTypeJackett:
				client, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "id", indexer.GetCompositeId())
					continue
				}
				clientById[indexer.GetCompositeId()] = client
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
			}
		}

		worker_queue.TorznabIndexerSyncerQueue.Process(func(item worker_queue.TorznabIndexerSyncerQueueItem) error {
			meta, nsid, err := getQueryMeta(log, item.SId)
			if err != nil {
				log.Error("failed to get query metadata", "error", err, "sid", item.SId)
				return nil
			}
			if len(meta.titles) == 0 {
				log.Debug("no titles found for stream", "sid", item.SId)
				return nil
			}

			for i := range indexers {
				indexer := &indexers[i]

				client, ok := clientById[indexer.GetCompositeId()]
				if !ok {
					continue
				}

				queriesBySid, err := buildQueriesForIndexer(client, nsid, meta)
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
					count := len(queries)
					err = torznab_indexer_syncinfo.Queue(indexer.Type, indexer.Id, sid, queries)
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
