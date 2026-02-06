package stremio_newz

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	newznabcache "github.com/MunifTanjim/stremthru/internal/newznab/cache"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	znabsearch "github.com/MunifTanjim/stremthru/internal/znab/search"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var streamTemplate = stremio_transformer.StreamTemplateDefault

type WrappedStream struct {
	*stremio.Stream
	R              *stremio_transformer.StreamExtractorResult
	nzbURL         string
	lockedDownload bool
	lockedProvider string
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

func matchesTitle(titles []string, parsedTitle string, normalizer *util.StringNormalizer) bool {
	for _, title := range titles {
		if util.MaxLevenshteinDistance(5, parsedTitle, title, normalizer) {
			return true
		}
	}
	return false
}

type indexerSearchQuery struct {
	indexer *Indexer
	znabsearch.IndexerQuery
}

func GetStreamsFromIndexers(reqCtx context.Context, ctx *Ctx, stremType, stremId string) ([]WrappedStream, error) {
	if len(ctx.Indexers) == 0 {
		return []WrappedStream{}, nil
	}

	log := ctx.Log

	queryMeta, nsid, err := znabsearch.GetQueryMeta(ctx.Log, stremId)
	if err != nil {
		return nil, err
	}

	sQueries := make([]indexerSearchQuery, 0, len(ctx.Indexers)*2)
	for i := range ctx.Indexers {
		indexer := ctx.Indexers[i]
		queriesBySId, err := znabsearch.BuildQueriesForNewznab(indexer, znabsearch.QueryBuilderConfig{
			Meta: queryMeta,
			NSId: nsid,
		})
		if err != nil {
			log.Error("failed to create search query", "error", err, "indexer", indexer.GetId())
			continue
		}
		for _, queries := range queriesBySId {
			for j := range queries {
				sQueries = append(sQueries, indexerSearchQuery{
					indexer:      &indexer,
					IndexerQuery: queries[j],
				})
			}
		}
	}

	type searchResult struct {
		indexer  *Indexer
		items    []newznab_client.Newz
		err      error
		is_exact bool
	}

	resultCh := make(chan searchResult, len(sQueries))

	for i := range sQueries {
		go func(sq indexerSearchQuery) {
			items, err := newznabcache.Search.Do(sq.indexer, sq.Query, sq.Header, log)
			resultCh <- searchResult{indexer: sq.indexer, items: items, err: err, is_exact: sq.IsExact}
		}(sQueries[i])
	}

	strn := util.NewStringNormalizer()
	wrappedStreams := []WrappedStream{}

	processItems := func(indexer *Indexer, items []newznab_client.Newz, is_exact bool) {
		for j := range items {
			item := &items[j]

			if item.GUID == "" || item.Size == 0 {
				continue
			}

			pttr, err := util.ParseTorrentTitle(item.Title)
			if err != nil {
				continue
			}

			if !is_exact {
				if !matchesTitle(queryMeta.Titles, pttr.Title, strn) {
					continue
				}
				if nsid.IsSeries() {
					if len(pttr.Seasons) > 0 && !slices.Contains(pttr.Seasons, queryMeta.Season) {
						continue
					}
					if len(pttr.Episodes) > 0 && !slices.Contains(pttr.Episodes, queryMeta.Ep) {
						continue
					}
				} else if queryMeta.Year > 0 {
					if pttr.Year != "" && pttr.Year != strconv.Itoa(queryMeta.Year) {
						continue
					}
				}
			}

			indexerName := item.Indexer.Name
			if indexerName == "" {
				indexerName = indexer.Name
			}

			data := &stremio_transformer.StreamExtractorResult{
				Result: pttr,

				Addon: stremio_transformer.StreamExtractorResultAddon{
					Name: "Newz",
				},
				Age:      item.Age(),
				Category: stremType,
				Hash:     item.GetHash(),
				Indexer:  indexerName,
				TTitle:   item.Title,
			}
			if item.Size > 0 {
				data.Size = util.ToSize(item.Size)
			}

			wrappedStreams = append(wrappedStreams, WrappedStream{
				lockedDownload: item.LockedDownload,
				lockedProvider: item.LockedProvider,
				nzbURL:         item.DownloadLink,
				R:              data,
				Stream: &stremio.Stream{
					Name:        data.Addon.Name,
					Description: data.TTitle,
					BehaviorHints: &stremio.StreamBehaviorHints{
						BingeGroup: "newz:" + data.Hash,
					},
				},
			})
		}
	}

	collected := 0
	for collected < len(sQueries) {
		select {
		case res := <-resultCh:
			collected++
			if res.err == nil && len(res.items) > 0 {
				processItems(res.indexer, res.items, res.is_exact)
			}
		case <-reqCtx.Done():
			log.Warn("indexer search timeout, returning partial results", "collected", collected, "total", len(sQueries))
			return wrappedStreams, nil
		}
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

	log := server.GetReqCtx(r).Log

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

	_, err = torrent_stream.NormalizeStreamId(id)
	if err != nil {
		if !errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
			log.Error("failed to normalize strem id", "error", err, "id", id)
			shared.ErrorInternalServerError(r, "failed to normalize strem id").WithCause(err).Send(w, r)
			return
		}
		shared.ErrorBadRequest(r, "unsupported strem id: "+id).Send(w, r)
		return
	}

	timeoutCtx, cancel := context.WithTimeout(r.Context(), config.Stremio.Newz.IndexerMaxTimeout)
	defer cancel()

	wrappedStreams, err := GetStreamsFromIndexers(timeoutCtx, ctx, contentType, id)
	if err != nil {
		log.Error("failed to get streams from indexers", "error", err)
		SendError(w, r, err)
		return
	}
	log.Debug("fetched streams from indexers", "count", len(wrappedStreams))

	mode := ud.Mode
	includeStream := mode.Stream()
	includeDebrid := mode.Debrid() && ud.HasStores()

	var isCachedByHash map[string]string
	var hasErrByStoreCode *util.Set[string]
	var checkNewzError error
	if includeDebrid && len(wrappedStreams) > 0 {
		hashes := make([]string, 0, len(wrappedStreams))
		for i := range wrappedStreams {
			stream := &wrappedStreams[i]
			hashes = append(hashes, stream.R.Hash)
		}
		res := ud.CheckNewz(&store.CheckNewzParams{
			Hashes: hashes,
		}, ctx.Log)
		if res.HasErr && len(res.ByHash) == 0 {
			checkNewzError = errors.Join(res.Err...)
		} else {
			isCachedByHash = res.ByHash
			hasErrByStoreCode = res.HasErrByStoreCode
		}
	}

	if checkNewzError != nil {
		SendError(w, r, checkNewzError)
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

	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/newz", eud, "playback", id)

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}

	for _, wStream := range wrappedStreams {
		hash := wStream.R.Hash

		nzbUrl := util.Base64Encode(wStream.nzbURL)
		if includeStream && !wStream.lockedDownload {
			wStream.R.Store.Code = ""
			wStream.R.Store.Name = ""
			wStream.R.Store.IsCached = false
			wStream.R.Store.IsProxied = false
			wasPreviouslySelected := shared.IsNZBCached(wStream.R.Hash)
			stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
			if err != nil {
				SendError(w, r, err)
				return
			}
			if wasPreviouslySelected {
				stream.Name = "📎 " + stream.Name
			}
			steamUrl := streamBaseUrl.JoinPath(string(UserDataModeStream), "st", url.PathEscape(nzbUrl), "/")
			if wStream.R.TTitle != "" {
				steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.TTitle))
			}
			stream.URL = steamUrl.String()
			if wasPreviouslySelected {
				cachedStreams = append(cachedStreams, *stream)
			} else {
				uncachedStreams = append(uncachedStreams, *stream)
			}
		}
		if includeDebrid {
			lockedProvider := strings.ToLower(wStream.lockedProvider)

			if storeCode, isCached := isCachedByHash[hash]; isCached && storeCode != "" {
				storeName := store.StoreCode(strings.ToLower(storeCode)).Name()

				if lockedProvider != "" && lockedProvider != string(storeName) {
					continue
				}

				wStream.R.Store.Code = storeCode
				wStream.R.Store.Name = string(storeName)
				wStream.R.Store.IsCached = true
				wStream.R.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
				stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
				if err != nil {
					SendError(w, r, err)
					return
				}
				escapedNZBURL := url.PathEscape(nzbUrl)
				if wStream.lockedDownload {
					escapedNZBURL = "-" + escapedNZBURL
				}
				steamUrl := streamBaseUrl.JoinPath(string(UserDataModeDebrid), strings.ToLower(storeCode), escapedNZBURL, "/")
				if wStream.R.TTitle != "" {
					steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.TTitle))
				}
				stream.URL = steamUrl.String()
				cachedStreams = append(cachedStreams, *stream)
			} else if !ud.CachedOnly {
				stores := ud.GetStores()
				for i := range stores {
					s := &stores[i]
					storeName := s.Store.GetName()

					if lockedProvider != "" && lockedProvider != string(storeName) {
						continue
					}

					storeCode := storeName.Code()
					if hasErrByStoreCode.Has(strings.ToUpper(string(storeCode))) {
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

					escapedNZBURL := url.PathEscape(nzbUrl)
					if wStream.lockedDownload {
						escapedNZBURL = "-" + escapedNZBURL
					}
					steamUrl := streamBaseUrl.JoinPath(string(UserDataModeDebrid), string(storeCode), escapedNZBURL, "/")
					if wStream.R.TTitle != "" {
						steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.TTitle))
					}
					stream.URL = steamUrl.String()
					uncachedStreams = append(uncachedStreams, *stream)
				}
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
