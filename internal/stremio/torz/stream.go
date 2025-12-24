package stremio_torz

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/alitto/pond/v2"
)

var streamTemplate = stremio_transformer.StreamTemplateDefault
var torzLazyPull = config.Stremio.Torz.LazyPull

type WrappedStream struct {
	*stremio.Stream
	R           *stremio_transformer.StreamExtractorResult
	torrentLink string
}

func (s WrappedStream) IsSortable() bool {
	return s.R != nil
}

func (s WrappedStream) GetQuality() string {
	return s.R.Quality
}

func (s WrappedStream) GetResolution() string {
	return s.R.Resolution
}

func (s WrappedStream) GetSize() string {
	return s.R.Size
}

func (s WrappedStream) GetHDR() string {
	return strings.Join(s.R.HDR, "|")
}

type indexerSearchQueryMeta struct {
	titles     []string
	year       int
	season, ep int
}

func (m *indexerSearchQueryMeta) MatchesTitle(parsedTitle string, normalizer *util.StringNormalizer) bool {
	for _, title := range m.titles {
		if util.MaxLevenshteinDistance(5, parsedTitle, title, normalizer) {
			return true
		}
	}
	return false
}

type indexerSearchQuery struct {
	indexer  tznc.Indexer
	query    *tznc.Query
	is_exact bool
}

var torrentFetchPool = pond.NewPool(20)

func GetStreamsFromIndexers(ctx *RequestContext, stremType, stremId string) ([]WrappedStream, []string, error) {
	if len(ctx.Indexers) == 0 {
		return []WrappedStream{}, []string{}, nil
	}

	log = ctx.Log

	nsid, err := torrent_stream.NormalizeStreamId(stremId)
	if err != nil {
		return nil, nil, err
	}

	queryMeta := indexerSearchQueryMeta{titles: []string{}}
	if nsid.IsAnime {
		if aniEp := util.SafeParseInt(nsid.Episode, -1); aniEp != -1 {
			tvdbMaps, err := anidb.GetTVDBEpisodeMaps(nsid.Id, false)
			if err != nil {
				return nil, nil, err
			}
			if epMap := tvdbMaps.GetByAnidbEpisode(aniEp); epMap != nil {
				ep := epMap.GetTMDBEpisode(aniEp)
				titles, err := anidb.GetTitlesByIds([]string{nsid.Id})
				if err != nil {
					return nil, nil, err
				}
				queryMeta.titles = make([]string, len(titles))
				queryMeta.season = epMap.TVDBSeason
				queryMeta.ep = ep
				seenTitle := util.NewSet[string]()
				for i := range titles {
					title := &titles[i]
					if seenTitle.Has(title.Value) {
						continue
					}
					seenTitle.Add(title.Value)
					queryMeta.titles[i] = title.Value
					if queryMeta.year == 0 && title.Year != "" {
						queryMeta.year = util.SafeParseInt(title.Year, 0)
					}
				}
			}
		}
	} else {
		it, err := imdb_title.Get(nsid.Id)
		if err != nil {
			return nil, nil, err
		}
		if it == nil {
			return nil, nil, errors.New("imdb title not found: " + nsid.Id)
		}
		queryMeta.titles = append(queryMeta.titles, it.Title)
		if it.Year > 0 {
			queryMeta.year = it.Year
		}
		if nsid.IsSeries() {
			queryMeta.season = util.SafeParseInt(nsid.Season, 0)
			queryMeta.ep = util.SafeParseInt(nsid.Episode, 0)
		}
		if it.OrigTitle != "" && it.OrigTitle != it.Title {
			queryMeta.titles = append(queryMeta.titles, it.OrigTitle)
		}
	}

	sQueries := make([]indexerSearchQuery, 0, len(ctx.Indexers)*2)
	for i := range ctx.Indexers {
		indexer := ctx.Indexers[i]

		query, err := indexer.NewSearchQuery(func(caps tznc.Caps) tznc.Function {
			if nsid.IsSeries() && caps.SupportsFunction(tznc.FunctionSearchTV) {
				return tznc.FunctionSearchTV
			}
			if caps.SupportsFunction(tznc.FunctionSearchMovie) {
				return tznc.FunctionSearchMovie
			}
			return tznc.FunctionSearch
		})
		if err != nil {
			log.Error("failed to create search query", "error", err, "indexer", indexer.GetId())
			continue
		}
		query.SetLimit(-1)
		if !nsid.IsAnime && query.IsSupported(tznc.SearchParamIMDBId) {
			query.Set(tznc.SearchParamIMDBId, nsid.Id)
			is_exact := !nsid.IsSeries()
			if nsid.IsSeries() {
				if query.IsSupported(tznc.SearchParamSeason) && nsid.Season != "" {
					query.Set(tznc.SearchParamSeason, nsid.Season)
					if query.IsSupported(tznc.SearchParamEp) && nsid.Episode != "" {
						query.Set(tznc.SearchParamEp, nsid.Episode)
						is_exact = true
					}
				}
			}
			sQueries = append(sQueries, indexerSearchQuery{
				indexer:  indexer,
				query:    query,
				is_exact: is_exact,
			})
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
					sQueries = append(sQueries, indexerSearchQuery{
						indexer: indexer,
						query:   query.Clone().Set(tznc.SearchParamQ, q.String()),
					})
					if queryMeta.season > 0 {
						q.WriteString(" S")
						q.WriteString(util.ZeroPadInt(queryMeta.season, 2))
						if queryMeta.ep > 0 {
							sQueries = append(sQueries, indexerSearchQuery{
								indexer: indexer,
								query:   query.Clone().Set(tznc.SearchParamQ, q.String()),
							})
							q.WriteString("E")
							q.WriteString(util.ZeroPadInt(queryMeta.ep, 2))
						}
						sQueries = append(sQueries, indexerSearchQuery{
							indexer: indexer,
							query:   query.Clone().Set(tznc.SearchParamQ, q.String()),
						})
					}
				} else if queryMeta.year > 0 {
					if !supportsYear {
						q.WriteString(" ")
						q.WriteString(strconv.Itoa(queryMeta.year))
					}
					sQueries = append(sQueries, indexerSearchQuery{
						indexer: indexer,
						query:   query.Clone().Set(tznc.SearchParamQ, q.String()),
					})
				}
			}
		}
	}

	var wg sync.WaitGroup
	results := make([][]tznc.Torz, len(sQueries))
	errs := make([]error, len(sQueries))
	for i := range sQueries {
		wg.Add(1)
		go func(sq indexerSearchQuery, i int) {
			defer wg.Done()
			start := time.Now()
			results[i], errs[i] = sq.indexer.Search(sq.query)
			if errs[i] == nil {
				log.Debug("indexer search completed", "indexer", sq.indexer.GetId(), "query", sq.query.Encode(), "duration", time.Since(start).String(), "count", len(results[i]))
			} else {
				log.Error("indexer search failed", "error", errs[i], "indexer", sq.indexer.GetId(), "query", sq.query.Encode(), "duration", time.Since(start).String())
			}
		}(sQueries[i], i)
	}
	wg.Wait()

	if len(results) == 0 && len(errs) > 0 {
		return nil, nil, errors.Join(errs...)
	}

	seenSourceURL := util.NewSet[string]()
	torzFetchWg := torrentFetchPool.NewGroup()
	for _, items := range results {
		for i := range items {
			item := &items[i]
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
	}
	err = torzFetchWg.Wait()
	if err != nil {
		log.Warn("errors occurred while fetching torrent magnets", "error", err)
	}

	hashSet := util.NewSet[string]()
	hashes := []string{}
	for _, items := range results {
		for i := range items {
			item := &items[i]
			if item.HasMissingData() {
				continue
			}
			if !hashSet.Has(item.Hash) {
				hashSet.Add(item.Hash)
				hashes = append(hashes, item.Hash)
			}
		}
	}

	tInfoByHash, err := torrent_info.GetByHashes(hashes)
	if err != nil {
		return nil, nil, err
	}

	filesByHashes, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		return nil, nil, err
	}

	strn := util.NewStringNormalizer()
	tInfosToUpsert := []torrent_info.TorrentItem{}
	wrappedStreams := []WrappedStream{}
	for i, items := range results {
		is_exact := sQueries[i].is_exact
		for i := range items {
			item := &items[i]
			if item.HasMissingData() {
				continue
			}

			if !is_exact {
				pttr, err := util.ParseTorrentTitle(item.Title)
				if err != nil {
					continue
				}
				if !queryMeta.MatchesTitle(pttr.Title, strn) {
					continue
				}
				if nsid.IsSeries() {
					if !slices.Contains(pttr.Seasons, queryMeta.season) {
						continue
					}
					if len(pttr.Episodes) > 0 && !slices.Contains(pttr.Episodes, queryMeta.ep) {
						continue
					}
				} else if queryMeta.year > 0 {
					if pttr.Year != "" && pttr.Year != strconv.Itoa(queryMeta.year) {
						continue
					}
				}
			}

			if tInfo, ok := tInfoByHash[item.Hash]; ok {
				var file *torrent_stream.File
				if files, ok := filesByHashes[item.Hash]; ok {
					idToMatch := stremId
					if idToMatch != "" {
						for i := range files {
							f := &files[i]
							if f.IsVideo() {
								if f.SId == idToMatch || f.ASId == idToMatch {
									file = f
								}
							}
						}
					}
				}
				fName := ""
				fIdx := -1
				fSize := int64(0)
				fVideoHash := ""
				if file != nil {
					fIdx = file.Idx
					fName = file.Name
					if file.Size > 0 {
						fSize = file.Size
					}
					fVideoHash = file.VideoHash
				} else if core.HasVideoExtension(tInfo.TorrentTitle) {
					fName = tInfo.TorrentTitle
				}

				pttr, err := tInfo.ToParsedResult()
				if err != nil {
					return nil, nil, err
				}

				if len(tInfo.TorrentTitle) < len(item.Title)/2 {
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

				data := &stremio_transformer.StreamExtractorResult{
					Hash:      tInfo.Hash,
					TTitle:    tInfo.TorrentTitle,
					Result:    pttr,
					Seeders:   item.Seeders,
					IsPrivate: tInfo.Private || item.Private,
					Indexer:   jackett.GetIndexerName(item.Indexer),
					Addon: stremio_transformer.StreamExtractorResultAddon{
						Name: "Torz",
					},
					Category: stremType,
					File: stremio_transformer.StreamExtractorResultFile{
						Name: fName,
						Idx:  fIdx,
					},
				}
				if fSize > 0 {
					data.File.Size = util.ToSize(fSize)
				}
				wrappedStreams = append(wrappedStreams, WrappedStream{
					torrentLink: item.SourceLink,
					R:           data,
					Stream: &stremio.Stream{
						Name:        data.Addon.Name,
						Description: data.TTitle,
						InfoHash:    data.Hash,
						FileIndex:   fIdx,
						BehaviorHints: &stremio.StreamBehaviorHints{
							Filename:   data.File.Name,
							VideoSize:  fSize,
							BingeGroup: "torz:" + data.Hash,
							VideoHash:  fVideoHash,
						},
					},
				})

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

			pttr, err := util.ParseTorrentTitle(tInfo.TorrentTitle)
			if err != nil {
				return nil, nil, err
			}

			data := &stremio_transformer.StreamExtractorResult{
				Hash:      item.Hash,
				TTitle:    item.Title,
				Seeders:   item.Seeders,
				IsPrivate: item.Private,
				Result:    pttr,
				Indexer:   jackett.GetIndexerName(item.Indexer),
				Addon: stremio_transformer.StreamExtractorResultAddon{
					Name: "Torz",
				},
				Category: stremType,
			}

			wrappedStream := WrappedStream{
				torrentLink: item.SourceLink,
				R:           data,
				Stream: &stremio.Stream{
					Name:        data.Addon.Name,
					Description: data.TTitle,
					InfoHash:    data.Hash,
					BehaviorHints: &stremio.StreamBehaviorHints{
						Filename:   data.File.Name,
						BingeGroup: "torz:" + data.Hash,
					},
				},
			}

			if len(item.Files) > 0 {
				var file *torrent_stream.File
				if files, ok := filesByHashes[item.Hash]; ok {
					idToMatch := stremId
					if nsid.IsAnime {
						if nsid.Id == "" {
							idToMatch = ""
						} else {
							idToMatch = nsid.Id + ":" + nsid.Episode
						}
					}
					if idToMatch != "" {
						for i := range files {
							f := &files[i]
							if f.IsVideo() {
								if f.SId == idToMatch || f.ASId == idToMatch {
									file = f
								}
							}
						}
					}
				}
				if file != nil {
					data.File.Idx = file.Idx
					wrappedStream.Stream.FileIndex = file.Idx
					data.File.Name = file.Name
					if file.Size > 0 {
						data.File.Size = util.ToSize(file.Size)
						wrappedStream.Stream.BehaviorHints.VideoSize = file.Size
					}
					wrappedStream.Stream.BehaviorHints.VideoHash = file.VideoHash
				} else if core.HasVideoExtension(tInfo.TorrentTitle) {
					data.File.Name = tInfo.TorrentTitle
				}
			}

			wrappedStreams = append(wrappedStreams, wrappedStream)
		}
	}
	go torrent_info.Upsert(tInfosToUpsert, torrent_info.TorrentInfoCategoryUnknown, false)

	return wrappedStreams, hashes, nil
}

func GetStreamsForHashes(stremType, stremId string, hashes []string, nsid *torrent_stream.NormalizedStremId) ([]WrappedStream, error) {
	tInfoByHash, err := torrent_info.GetByHashes(hashes)
	if err != nil {
		return nil, err
	}

	filesByHashes, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		return nil, err
	}

	wrappedStreams := make([]WrappedStream, 0, len(hashes))
	for _, hash := range hashes {
		tInfo, ok := tInfoByHash[hash]
		if !ok {
			continue
		}

		var file *torrent_stream.File
		if files, ok := filesByHashes[hash]; ok {
			idToMatch := stremId
			if nsid.IsAnime {
				if nsid.Id == "" {
					idToMatch = ""
				} else {
					idToMatch = nsid.Id + ":" + nsid.Episode
				}
			}
			if idToMatch != "" {
				for i := range files {
					f := &files[i]
					if f.IsVideo() {
						if f.SId == idToMatch || f.ASId == idToMatch {
							file = f
							break
						}
					}
				}
			}
			if file == nil && !nsid.IsAnime && nsid.IsSeries() && nsid.Episode != "" {
				for i := range files {
					f := &files[i]
					if f.IsVideo() {
						data := stremio_shared.ParseSeasonEpisodeFromName(f.Name, false)
						if data.Season != -1 && strconv.Itoa(data.Season) != nsid.Season {
							continue
						}
						if data.Episode == -1 || strconv.Itoa(data.Episode) != nsid.Episode {
							continue
						}
						file = f
						break
					}
				}
				if file == nil {
					log.Trace("skipping torrent: no matching file found", "hash", tInfo.Hash, "sid", stremId)
					continue
				}
			}
		}

		fName := ""
		fIdx := -1
		fSize := int64(0)
		fVideoHash := ""
		if file != nil {
			fIdx = file.Idx
			fName = file.Name
			if file.Size > 0 {
				fSize = file.Size
			}
			fVideoHash = file.VideoHash
		} else if core.HasVideoExtension(tInfo.TorrentTitle) {
			fName = tInfo.TorrentTitle
		}

		pttr, err := tInfo.ToParsedResult()
		if err != nil {
			return nil, err
		}
		data := &stremio_transformer.StreamExtractorResult{
			Hash:      tInfo.Hash,
			TTitle:    tInfo.TorrentTitle,
			Result:    pttr,
			Seeders:   tInfo.Seeders,
			IsPrivate: tInfo.Private,
			Indexer:   jackett.GetIndexerName(tInfo.Indexer),
			Addon: stremio_transformer.StreamExtractorResultAddon{
				Name: "Torz",
			},
			Category: stremType,
			File: stremio_transformer.StreamExtractorResultFile{
				Name: fName,
				Idx:  fIdx,
			},
		}
		if fSize > 0 {
			data.File.Size = util.ToSize(fSize)
		}
		wrappedStreams = append(wrappedStreams, WrappedStream{
			R: data,
			Stream: &stremio.Stream{
				Name:        data.Addon.Name,
				Description: data.TTitle,
				InfoHash:    data.Hash,
				FileIndex:   fIdx,
				BehaviorHints: &stremio.StreamBehaviorHints{
					Filename:   data.File.Name,
					VideoSize:  fSize,
					BingeGroup: "torz:" + data.Hash,
					VideoHash:  fVideoHash,
				},
			},
		})
	}
	return wrappedStreams, nil
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		shared.ErrorBadRequest(r, "failed to get request context: "+err.Error()).Send(w, r)
		return
	}

	contentType := r.PathValue("contentType")
	id := stremio_shared.GetPathValue(r, "id")

	isImdbId := strings.HasPrefix(id, "tt")
	isKitsuId := strings.HasPrefix(id, "kitsu:")
	isMALId := strings.HasPrefix(id, "mal:")
	isAnime := isKitsuId || isMALId

	if isImdbId {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else if isAnime {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) && contentType != "anime" {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w, r)
		return
	}

	eud := ud.GetEncoded()

	pulledHashes := []string{}
	nsid, err := torrent_stream.NormalizeStreamId(id)
	if err == nil {
		cleanSId := nsid.ToClean()
		if torzLazyPull {
			go buddy.PullTorrentsByStremId(cleanSId, "")
		} else {
			hashes := buddy.PullTorrentsByStremId(cleanSId, "")
			pulledHashes = append(pulledHashes, hashes...)
		}

		if !nsid.IsAnime {
			worker_queue.TorznabIndexerSyncerQueue.Queue(worker_queue.TorznabIndexerSyncerQueueItem{
				SId: nsid.String(),
			})
		}
	} else if !errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
		log.Error("failed to normalize strem id", "error", err, "id", id)
		shared.ErrorInternalServerError(r, "failed to normalize strem id").WithCause(err).Send(w, r)
		return
	} else {
		shared.ErrorBadRequest(r, "unsupported strem id: "+id).Send(w, r)
		return
	}

	hashes, err := torrent_info.ListHashesByStremId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	hashSet := util.NewSet[string]()
	for _, hash := range hashes {
		hashSet.Add(hash)
	}

	if len(pulledHashes) > 0 {
		for _, hash := range pulledHashes {
			if !hashSet.Has(hash) {
				hashSet.Add(hash)
				hashes = append(hashes, hash)
			}
		}
	}

	var wg sync.WaitGroup

	var wrappedStreams []WrappedStream
	var getStreamsError error
	wg.Go(func() {
		wrappedStreams, getStreamsError = GetStreamsForHashes(contentType, id, hashes, nsid)
	})

	var wrappedStreamsFromIndexers []WrappedStream
	var hashesFromIndexers []string
	var getStreamsFromIndexersError error
	wg.Go(func() {
		var streams []WrappedStream
		var hashes []string
		var err error
		done := make(chan struct{})
		go func() {
			streams, hashes, err = GetStreamsFromIndexers(ctx, contentType, id)
			close(done)
		}()

		select {
		case <-done:
			wrappedStreamsFromIndexers = streams
			hashesFromIndexers = hashes
			getStreamsFromIndexersError = err
			log.Debug("fetched streams from indexers", "count", len(streams))
		case <-time.After(config.Stremio.Torz.IndexerMaxTimeout):
			log.Warn("fetching streams from indexers timed out")
		}
	})

	wg.Wait()

	if getStreamsError != nil {
		SendError(w, r, getStreamsError)
		return
	}

	if getStreamsFromIndexersError != nil {
		log.Error("failed to get streams from indexers", "error", getStreamsFromIndexersError)
	} else {
		wrappedStreams = append(wrappedStreams, wrappedStreamsFromIndexers...)
		for _, hash := range hashesFromIndexers {
			if !hashSet.Has(hash) {
				hashSet.Add(hash)
				hashes = append(hashes, hash)
			}
		}
	}

	isP2P := ud.IsP2P()

	var isCachedByHash map[string]string
	var hasErrByStoreCode map[string]struct{}
	var checkMagnetError error
	if !isP2P && len(hashes) > 0 {
		cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
			Magnets:  hashes,
			ClientIP: ctx.ClientIP,
			SId:      id,
		}, ctx.Log)
		if cmRes.HasErr && len(cmRes.ByHash) == 0 {
			checkMagnetError = errors.Join(cmRes.Err...)
		} else {
			isCachedByHash = cmRes.ByHash
			hasErrByStoreCode = cmRes.HasErrByStoreCode
		}
	}

	if checkMagnetError != nil {
		SendError(w, r, checkMagnetError)
		return
	}

	if ud.Filter != "" {
		filter, err := stremio_transformer.StreamFilterBlob(ud.Filter).Parse()
		if err == nil {
			wrappedStreams = filterStreams(wrappedStreams, filter)
		} else {
			log.Warn("failed to parse filter expression", "error", err)
		}
	}

	stremio_transformer.SortStreams(wrappedStreams, ud.Sort)

	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/torz", eud, "_/strem", id)

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}
	for _, wStream := range wrappedStreams {
		hash := wStream.R.Hash
		if isP2P {
			if wStream.FileIndex == -1 || wStream.R.IsPrivate {
				continue
			}

			wStream.R.Store.Code = "P2P"
			wStream.R.Store.Name = "P2P"
			stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
			if err != nil {
				SendError(w, r, err)
				return
			}
			uncachedStreams = append(uncachedStreams, *stream)
		} else if storeCode, isCached := isCachedByHash[hash]; isCached && storeCode != "" {
			storeName := store.StoreCode(strings.ToLower(storeCode)).Name()
			wStream.R.Store.Code = storeCode
			wStream.R.Store.Name = string(storeName)
			wStream.R.Store.IsCached = true
			wStream.R.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
			stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
			if err != nil {
				SendError(w, r, err)
				return
			}
			identifier := hash
			steamUrl := streamBaseUrl.JoinPath(strings.ToLower(storeCode), identifier, strconv.Itoa(wStream.R.File.Idx), "/")
			if wStream.R.File.Name != "" {
				steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.File.Name))
			}
			stream.URL = steamUrl.String()
			stream.InfoHash = ""
			stream.FileIndex = 0
			cachedStreams = append(cachedStreams, *stream)
		} else if !ud.CachedOnly {
			stores := ud.GetStores()
			for i := range stores {
				s := &stores[i]
				storeName := s.Store.GetName()
				storeCode := storeName.Code()
				if _, hasErr := hasErrByStoreCode[strings.ToUpper(string(storeCode))]; hasErr || storeCode == store.StoreCodeEasyDebrid {
					continue
				}
				if wStream.R.IsPrivate && (!ud.IncludeUncachedPrivate || wStream.torrentLink == "" || storeCode != store.StoreCodeTorBox) {
					continue
				}

				origStream := *wStream.Stream
				wStream.R.Store.Code = strings.ToUpper(string(storeCode))
				wStream.R.Store.Name = string(storeName)
				wStream.R.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
				stream, err := streamTemplate.Execute(&origStream, wStream.R)
				if err != nil {
					SendError(w, r, err)
					return
				}

				identifier := hash
				if wStream.R.IsPrivate {
					identifier += "-" + core.Base64Encode(wStream.torrentLink)
				}
				steamUrl := streamBaseUrl.JoinPath(string(storeCode), identifier, strconv.Itoa(wStream.R.File.Idx), "/")
				if wStream.R.File.Name != "" {
					steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.File.Name))
				}
				stream.URL = steamUrl.String()
				stream.InfoHash = ""
				stream.FileIndex = 0
				uncachedStreams = append(uncachedStreams, *stream)
			}
		}
	}

	streams := make([]stremio.Stream, len(cachedStreams)+len(uncachedStreams))
	idx := 0
	for i := range cachedStreams {
		streams[idx] = cachedStreams[i]
		idx++
	}
	for i := range uncachedStreams {
		streams[idx] = uncachedStreams[i]
		idx++
	}

	if isP2P && !torzLazyPull {
		w.Header().Set("Cache-Control", "public, max-age=7200")
	}

	SendResponse(w, r, 200, &stremio.StreamHandlerResponse{
		Streams: streams,
	})
}

func filterStreams(streams []WrappedStream, filter *stremio_transformer.StreamFilter) []WrappedStream {
	result := make([]WrappedStream, 0, len(streams))
	for i := range streams {
		stream := &streams[i]
		if stream.R == nil || filter.Match(stream.R) {
			result = append(result, *stream)
		}
	}
	return result
}
