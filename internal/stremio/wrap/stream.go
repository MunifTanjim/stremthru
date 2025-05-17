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
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

func (ud UserData) fetchStream(ctx *context.StoreContext, r *http.Request, rType, id string) (*stremio.StreamHandlerResponse, error) {
	log := ctx.Log

	eud := ud.GetEncoded()

	stremId := strings.TrimSuffix(id, ".json")

	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameStream, rType, id)
	if err != nil {
		return nil, err
	}
	upstreamsCount := len(upstreams)
	log.Debug("found addons for stream", "count", upstreamsCount)

	chunks := make([][]WrappedStream, upstreamsCount)
	errs := make([]error, upstreamsCount)

	template, err := ud.template.Parse()
	if err != nil {
		return nil, err
	}

	isImdbStremId := strings.HasPrefix(stremId, "tt")
	torrentInfoCategory := torrent_info.GetCategoryFromStremId(stremId)

	if isImdbStremId {
		go buddy.PullTorrentsByStremId(stremId, "")
	}

	var wg sync.WaitGroup
	for i := range upstreams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			up := &upstreams[i]
			res, err := addon.FetchStream(&stremio_addon.FetchStreamParams{
				BaseURL:  up.baseUrl,
				Type:     rType,
				Id:       id,
				ClientIP: ctx.ClientIP,
			})
			streams := res.Data.Streams
			wstreams := make([]WrappedStream, len(streams))
			errs[i] = err
			tInfos := []torrent_info.TorrentInfoInsertData{}
			if err == nil {
				extractor, err := up.extractor.Parse()
				if err != nil {
					errs[i] = err
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
			chunks[i] = wstreams
		}()
	}
	wg.Wait()

	allStreams := []WrappedStream{}
	for i := range chunks {
		if errs[i] != nil {
			hostname := upstreams[i].baseUrl.Hostname()
			log.Error("failed to fetch streams", "error", errs[i], "hostname", hostname)
		} else {
			allStreams = append(allStreams, chunks[i]...)
		}
	}

	if template != nil {
		stremio_transformer.SortStreams(allStreams, ud.Sort)
	}

	totalStreams := len(allStreams)
	allStreams = dedupeStreams(allStreams)
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
	if len(hashes) > 0 {
		cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
			Magnets:  hashes,
			ClientIP: ctx.ClientIP,
			SId:      stremId,
		}, log)
		if cmRes.HasErr {
			return nil, errors.Join(cmRes.Err...)
		}
		isCachedByHash = cmRes.ByHash
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
			} else if !ud.CachedOnly {
				surlRawQuery := surl.RawQuery
				stores := ud.GetStores()
				for i := range stores {
					s := &stores[i]
					storeName := s.Store.GetName()
					storeCode := strings.ToUpper(string(storeName.Code()))
					if storeCode == "ED" {
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
