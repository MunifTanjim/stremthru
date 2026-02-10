package stremio_wrap

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_torz "github.com/MunifTanjim/stremthru/internal/stremio/torz"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var lazyPullTorz = config.Stremio.Torz.LazyPull

func (ud UserData) fetchStream(ctx *Ctx, r *http.Request, rType, id string) (*stremio.StreamHandlerResponse, error) {
	log := ctx.Log

	eud := ud.GetEncoded()

	stremId := strings.TrimSuffix(id, ".json")

	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameStream, rType, id)
	if err != nil {
		return nil, err
	}
	upstreamsCount := len(upstreams)
	log.Debug("found addons for stream", "count", upstreamsCount)

	chunksCount := upstreamsCount
	if ud.IncludeTorz {
		chunksCount += 1
	}
	chunks := make([][]WrappedStream, chunksCount)
	errs := make([]error, chunksCount)

	template, err := ud.template.Parse()
	if err != nil {
		return nil, err
	}

	isImdbStremId := strings.HasPrefix(stremId, "tt")
	torrentInfoCategory := torrent_info.GetCategoryFromStremId(stremId, rType)

	nsid, err := torrent_stream.NormalizeStreamId(stremId)
	if err == nil {
		cleanSId := nsid.ToClean()
		if !ud.IncludeTorz || lazyPullTorz {
			go buddy.PullTorrentsByStremId(cleanSId, "")
		} else {
			buddy.PullTorrentsByStremId(cleanSId, "")
		}

		torznab_indexer_syncinfo.QueueJob(nsid.String())
	} else if !errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
		log.Error("failed to normalize strem id", "strem_id", stremId, "error", err)
	} else {
		log.Warn("unsupported strem id for normalization", "strem_id", stremId)
	}

	chunkIdxOffset := 0
	var wg sync.WaitGroup
	if ud.IncludeTorz {
		chunkIdxOffset = 1
		wg.Go(func() {

			hashes, err := torrent_info.ListHashesByStremId(stremId)
			if err != nil {
				if errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
					return
				}

				errs[0] = err
				return
			}

			if nsid == nil {
				return
			}

			streams, err := stremio_torz.GetStreamsForHashes(rType, stremId, hashes, nsid)
			if err != nil {
				errs[0] = err
				return
			}

			wstreams := make([]WrappedStream, len(streams))
			for i := range streams {
				wstream := &streams[i]
				stream := wstream.Stream
				tmpl := template
				if tmpl == nil || tmpl.IsEmpty() || tmpl.IsRaw() {
					tmpl = stremio_transformer.StreamTemplateDefault
				}
				s, err := tmpl.Execute(stream, wstream.R)
				if err != nil {
					errs[0] = err
					return
				}
				wstreams[i] = WrappedStream{
					Stream: s,
					r:      wstream.R,
				}
			}
			chunks[0] = wstreams
		})
	}
	for i := range upstreams {
		idx := i + chunkIdxOffset
		wg.Go(func() {
			up := &upstreams[i]
			res, err := addon.FetchStream(&stremio_addon.FetchStreamParams{
				BaseURL:  up.baseUrl,
				Type:     rType,
				Id:       id,
				ClientIP: ctx.ClientIP,
			})
			streams := res.Data.Streams
			wstreams := make([]WrappedStream, len(streams))
			errs[idx] = err
			tInfos := []torrent_info.TorrentInfoInsertData{}
			if err == nil {
				extractor, err := up.extractor.Parse()
				if err != nil {
					errs[idx] = err
				} else {
					addonHostname := up.baseUrl.Hostname()
					transformer := StreamTransformer{
						Extractor: extractor,
						Template:  template,
					}
					for i := range streams {
						stream := streams[i]
						if isImdbStremId {
							if cData := torrent_info.ExtractCreateDataFromStream(addonHostname, stremId, &stream); cData != nil {
								tInfos = append(tInfos, *cData)
							}
						}
						wstream, err := transformer.Do(&stream, rType, up.ReconfigureStore)
						if err != nil {
							LogError(r, "failed to transform stream", err)
						}
						if up.NoContentProxy {
							wstream.noContentProxy = true
						}
						wstreams[i] = *wstream
					}
				}
			}
			if isImdbStremId {
				if len(tInfos) > 0 {
					worker.TorrentPusherQueue.Queue(stremId)
				}
				go torrent_info.Upsert(tInfos, torrentInfoCategory, false)
			}
			chunks[idx] = wstreams
		})
	}
	wg.Wait()

	allStreams := []WrappedStream{}
	if ud.IncludeTorz {
		if errs[0] != nil {
			log.Error("failed to fetch torz streams", "error", errs[0])
		} else {
			allStreams = append(allStreams, chunks[0]...)
		}
	}
	for i := range chunks[chunkIdxOffset:] {
		idx := i + chunkIdxOffset
		hostname := upstreams[i].baseUrl.Hostname()
		if errs[idx] != nil {
			log.Error("failed to fetch streams", "error", errs[idx], "hostname", hostname)
		} else {
			allStreams = append(allStreams, chunks[idx]...)
		}
	}

	if ud.IncludeTorz {
		allStreams = dedupeStreams(allStreams)
	}

	if ud.Filter != "" {
		filter, err := stremio_transformer.StreamFilterBlob(ud.Filter).Parse()
		if err == nil {
			allStreams = filterStreams(allStreams, filter)
		} else {
			log.Warn("failed to parse filter expression", "error", err)
		}
	}

	if template != nil {
		stremio_transformer.SortStreams(allStreams, ud.Sort)
	}

	if !ud.IncludeTorz {
		allStreams = dedupeStreams(allStreams)
	}

	totalStreams := len(allStreams)
	log.Debug("found streams", "total_count", totalStreams, "deduped_count", len(allStreams))

	hashes := []string{}
	magnetByHash := map[string]core.MagnetLink{}
	for i := range allStreams {
		stream := &allStreams[i]
		if stream.URL == "" && stream.InfoHash != "" {
			magnet, err := core.ParseMagnetLink(stream.InfoHash)
			if err != nil {
				continue
			}
			hashes = append(hashes, magnet.Hash)
			magnetByHash[magnet.Hash] = magnet
		}
	}

	isCachedByHash := map[string]string{}
	hasErrByStoreCode := map[string]struct{}{}
	if len(hashes) > 0 {
		cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
			Magnets:  hashes,
			ClientIP: ctx.ClientIP,
			SId:      stremId,
		}, log)
		if cmRes.HasErr && len(cmRes.ByHash) == 0 {
			return nil, errors.Join(cmRes.Err...)
		}
		isCachedByHash = cmRes.ByHash
		hasErrByStoreCode = cmRes.HasErrByStoreCode
	}

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}
	for i := range allStreams {
		stream := &allStreams[i]
		if stream.URL == "" && stream.InfoHash != "" {
			magnet, ok := magnetByHash[strings.ToLower(stream.InfoHash)]
			if !ok {
				continue
			}
			surl := shared.ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/_/strem/" + magnet.Hash + "/" + strconv.Itoa(stream.FileIndex) + "/")
			if stream.BehaviorHints != nil && stream.BehaviorHints.Filename != "" {
				surl = surl.JoinPath(url.PathEscape(stream.BehaviorHints.Filename))
			}
			surl.RawQuery = "sid=" + stremId
			if stream.r != nil && stream.r.Season != -1 && stream.r.Episode != -1 {
				surl.RawQuery += "&re=" + url.QueryEscape(strconv.Itoa(stream.r.Season)+".{1,3}"+strconv.Itoa(stream.r.Episode))
			}
			stream.InfoHash = ""
			stream.FileIndex = 0

			storeCode, ok := isCachedByHash[magnet.Hash]
			if ok && storeCode != "" {
				surl.RawQuery += "&s=" + storeCode
				stream.URL = surl.String()
				stream.Name = "⚡ [" + storeCode + "] " + stream.Name

				if ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(ud.GetStoreByCode(storeCode).Store.GetName())) {
					stream.Name = "✨ " + stream.Name
				}

				cachedStreams = append(cachedStreams, *stream.Stream)
			} else if !ud.CachedOnly && (stream.r == nil || !stream.r.IsPrivate) {
				surlRawQuery := surl.RawQuery
				stores := ud.GetStores()
				for i := range stores {
					s := &stores[i]
					storeName := s.Store.GetName()
					storeCode := strings.ToUpper(string(storeName.Code()))
					if _, hasErr := hasErrByStoreCode[storeCode]; hasErr || storeCode == "ED" {
						continue
					}

					stream := *stream.Stream
					surl.RawQuery = surlRawQuery + "&s=" + storeCode
					stream.URL = surl.String()
					stream.Name = "[" + storeCode + "] " + stream.Name

					if ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.IsProxyAuthorized {
						stream.Name = "✨ " + stream.Name
					}

					uncachedStreams = append(uncachedStreams, stream)
				}
			}
		} else if stream.URL != "" {
			if !stream.noContentProxy {
				var headers map[string]string
				if stream.BehaviorHints != nil && stream.BehaviorHints.ProxyHeaders != nil && stream.BehaviorHints.ProxyHeaders.Request != nil {
					headers = stream.BehaviorHints.ProxyHeaders.Request
				}

				if ctx.IsProxyAuthorized {
					if url, err := shared.CreateProxyLink(r, stream.URL, headers, config.TUNNEL_TYPE_AUTO, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, ""); err == nil && url != stream.URL {
						stream.URL = url
						stream.Name = "✨ " + stream.Name
					}
				}
			}
			if stream.r == nil || stream.r.Store.IsCached {
				cachedStreams = append(cachedStreams, *stream.Stream)
			} else {
				uncachedStreams = append(uncachedStreams, *stream.Stream)
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

	return &stremio.StreamHandlerResponse{
		Streams: streams,
	}, nil
}

func filterStreams(streams []WrappedStream, filter *stremio_transformer.StreamFilter) []WrappedStream {
	result := make([]WrappedStream, 0, len(streams))
	for i := range streams {
		stream := &streams[i]
		if stream.r == nil || filter.Match(stream.r) {
			result = append(result, *stream)
		}
	}
	return result
}
