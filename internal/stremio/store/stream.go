package stremio_store

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

var streamTemplate = func() *stremio_transformer.StreamTemplate {
	tmplBlob := stremio_transformer.StreamTemplateDefault.Blob
	tmplBlob.Description = "✏️ {{.Title}}\n" + tmplBlob.Description
	tmpl := tmplBlob.MustParse()
	return tmpl
}()

type StreamFileMatcher struct {
	MagnetId       string
	FileLink       string
	FileName       string
	UseLargestFile bool
	Episode        int
	Season         int

	IdR        *ParsedId
	IdPrefix   string
	Store      store.Store
	StoreCode  string
	StoreToken string
	ClientIP   string
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

	videoIdWithLink := getId(r)
	contentType := r.PathValue("contentType")
	isStremThruStoreId := isStoreId(videoIdWithLink)
	isImdbId := strings.HasPrefix(videoIdWithLink, "tt")
	if isStremThruStoreId {
		if contentType != ContentTypeOther {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else if isImdbId {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	res := stremio.StreamHandlerResponse{
		Streams: []stremio.Stream{},
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	var meta *stremio.Meta
	season, episode := -1, -1

	matchers := []StreamFileMatcher{}

	if isStremThruStoreId {
		idr, err := parseId(videoIdWithLink)
		if err != nil {
			SendError(w, r, err)
			return
		}

		ctx, err := ud.GetRequestContext(r, idr)
		if err != nil || ctx.Store == nil {
			if err != nil {
				LogError(r, "failed to get request context", err)
			}
			shared.ErrorBadRequest(r, "").Send(w, r)
			return
		}

		idPrefix := getIdPrefix(idr.getStoreCode())
		videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
		videoId, escapedLink, _ := strings.Cut(videoId, ":")
		link, err := url.PathUnescape(escapedLink)
		if err != nil {
			LogError(r, "failed to parse link", err)
			SendError(w, r, err)
			return
		}

		matchers = append(matchers, StreamFileMatcher{
			MagnetId: videoId,
			FileLink: link,

			IdPrefix:   idPrefix,
			IdR:        idr,
			Store:      ctx.Store,
			StoreCode:  idr.getStoreCode(),
			StoreToken: ctx.StoreAuthToken,
			ClientIP:   ctx.ClientIP,
		})
	}

	if isImdbId {
		sType, sId := "", ""
		sType, sId, season, episode = parseStremId(videoIdWithLink)
		mres, err := fetchMeta(sType, sId, core.GetRequestIP(r))
		if err != nil {
			SendError(w, r, err)
			return
		}
		meta = &mres.Meta

		var wg sync.WaitGroup

		idPrefixes := ud.getIdPrefixes()
		errs := make([]error, len(idPrefixes))
		matcherResults := make([][]StreamFileMatcher, len(idPrefixes))

		for idx, idPrefix := range idPrefixes {
			wg.Add(1)
			go func() {
				defer wg.Done()

				idr, err := parseId(idPrefix)
				if err != nil {
					errs[idx] = err
					return
				}
				ctx, err := ud.GetRequestContext(r, idr)
				if err != nil || ctx.Store == nil {
					if err != nil {
						LogError(r, "failed to get request context", err)
					}
					errs[idx] = shared.ErrorBadRequest(r, "")
					return
				}

				items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, idPrefix, idr)
				if meta.Name != "" {
					query := strings.ToLower(meta.Name)
					filteredItems := []CachedCatalogItem{}
					for i := range items {
						item := &items[i]
						if fuzzy.TokenSetRatio(query, strings.ToLower(item.Name), false, true) > 90 {
							filteredItems = append(filteredItems, *item)
						}
					}
					items = filteredItems
				}

				for i := range items {
					item := &items[i]
					id := strings.TrimPrefix(item.Id, idPrefix)
					if sType == "series" {
						matcherResults[idx] = append(matcherResults[idx], StreamFileMatcher{
							MagnetId: id,
							Season:   season,
							Episode:  episode,

							IdPrefix:   idPrefix,
							IdR:        idr,
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
						})
					} else {
						matcherResults[idx] = append(matcherResults[idx], StreamFileMatcher{
							MagnetId:       id,
							UseLargestFile: true,

							IdPrefix:   idPrefix,
							IdR:        idr,
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
						})
					}
				}
			}()
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				SendError(w, r, err)
				return
			}
		}
		for i := range matcherResults {
			matchers = append(matchers, matcherResults[i]...)
		}
	}

	var wg sync.WaitGroup
	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/_/strem/")
	errs := make([]error, len(matchers))
	streams := make([]*stremio.Stream, len(matchers))
	for i, matcher := range matchers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cInfo, err := getStoreContentInfo(matcher.Store, matcher.StoreToken, matcher.MagnetId, matcher.ClientIP, matcher.IdR)
			if err != nil {
				errs[i] = err
				return
			}

			if !matcher.IdR.isUsenet && !matcher.IdR.isWebDL && meta == nil {
				stremIdByHash, err := torrent_stream.GetStremIdByHashes([]string{cInfo.Hash})
				if err != nil {
					log.Error("failed to get strem id by hashes", "error", err)
				}
				if stremId := stremIdByHash.Get(cInfo.Hash); stremId != "" {
					sType, sId, sSeason, sEpisode := parseStremId(stremId)
					if !isStremThruStoreId {
						season, episode = sSeason, sEpisode
					}
					if mRes, err := fetchMeta(sType, sId, core.GetRequestIP(r)); err == nil {
						meta = &mRes.Meta
					} else {
						log.Error("failed to fetch meta", "error", err)
					}
				}
			}

			tpttr, err := util.ParseTorrentTitle(cInfo.Name)
			if err != nil {
				pttLog.Warn("failed to parse", "error", err, "title", cInfo.Name)
			}
			tSeason, tEpisode := -1, -1
			if len(tpttr.Seasons) == 1 {
				tSeason = tpttr.Seasons[0]
			}
			if len(tpttr.Episodes) == 1 {
				tEpisode = tpttr.Episodes[0]
			}

			var pttr *ptt.Result
			var file *store.MagnetFile

			for i := range cInfo.Files {
				f := &cInfo.Files[i]
				if !core.HasVideoExtension(f.Name) {
					continue
				}

				if matcher.FileLink != "" && matcher.FileLink == f.Link {
					file = f
					break
				} else if matcher.FileName != "" && matcher.FileName == f.Name {
					file = f
					break
				} else if matcher.Episode > 0 {
					if r, err := util.ParseTorrentTitle(f.Name); err == nil {
						pttr = r
						fSeason, fEpisode := tSeason, tEpisode
						if len(r.Seasons) > 0 {
							fSeason = r.Seasons[0]
						} else if tSeason != -1 {
							r.Seasons = append(r.Seasons, tSeason)
						}
						if len(r.Episodes) > 0 {
							fEpisode = r.Episodes[0]
						} else if tEpisode != -1 {
							r.Episodes = append(r.Episodes, tEpisode)
						}
						if fSeason == matcher.Season && fEpisode == matcher.Episode {
							file = f
							season, episode = fSeason, fEpisode
							break
						}
					} else {
						pttLog.Warn("failed to parse", "error", err, "title", f.Name)
					}
					pttr = nil
				} else if matcher.UseLargestFile {
					if file == nil || file.Size < f.Size {
						file = f
					}
				}
			}

			if file == nil {
				return
			}

			streamId := matcher.IdPrefix + matcher.MagnetId + ":" + file.Link
			stream := stremio.Stream{
				URL:  streamBaseUrl.JoinPath(url.PathEscape(streamId)).String(),
				Name: file.Name,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: matcher.IdPrefix + cInfo.Hash,
					Filename:   file.Name,
					VideoSize:  file.Size,
				},
			}
			if pttr == nil {
				if r, err := util.ParseTorrentTitle(file.Name); err == nil {
					pttr = r
					if len(pttr.Seasons) > 0 {
						season = pttr.Seasons[0]
					} else {
						season = tSeason
					}
					if len(pttr.Episodes) > 0 {
						episode = pttr.Episodes[0]
					} else {
						episode = tEpisode
					}
				} else {
					pttLog.Warn("failed to parse", "error", err, "title", file.Name)
				}
			}
			if pttr != nil {
				if tpttr.Error() == nil {
					if pttr.Resolution == "" {
						pttr.Resolution = tpttr.Resolution
					}
					if pttr.Quality == "" {
						pttr.Quality = tpttr.Quality
					}
					if pttr.Codec == "" {
						pttr.Codec = tpttr.Codec
					}
					if len(pttr.HDR) == 0 {
						pttr.HDR = tpttr.HDR
					}
					if len(pttr.Audio) == 0 {
						pttr.Audio = tpttr.Audio
					}
					if len(pttr.Channels) == 0 {
						pttr.Channels = tpttr.Channels
					}
					if pttr.Group == "" {
						pttr.Group = tpttr.Group
					}
				}
				pttr.Size = util.ToSize(file.Size)
				if meta != nil && season != -1 && episode != -1 {
					for i := range meta.Videos {
						video := &meta.Videos[i]
						if video.Season.Equal(season) && video.Episode.Equal(episode) {
							pttr.Title = video.Name
							break
						}
					}
				}
				storeName := matcher.Store.GetName()
				data := &stremio_transformer.StreamExtractorResult{
					Result: pttr,
					File: stremio_transformer.StreamExtractorResultFile{
						Name: file.Name,
						Idx:  file.Idx,
						Size: pttr.Size,
					},
					Raw: stremio_transformer.StreamExtractorResultRaw{
						Name: stream.Name,
					},
					Store: stremio_transformer.StreamExtractorResultStore{
						Code:      strings.ToUpper(string(storeName.Code())),
						Name:      string(storeName),
						IsCached:  true,
						IsProxied: matcher.IdR.isST && config.StoreContentProxy.IsEnabled(string(storeName)),
					},
				}
				if matcher.IdR.isUsenet {
					data.Addon.Name = "Usenet Store"
				} else if matcher.IdR.isWebDL {
					data.Addon.Name = "WebDL Store"
				} else {
					data.Addon.Name = "Store"
				}
				if _, err := streamTemplate.Execute(&stream, data); err != nil {
					log.Error("failed to execute stream template", "error", err)
				}
			}

			streams[i] = &stream
		}()
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		log.Error("failed to get stream", "error", err)
	}

	for i := range streams {
		if streams[i] != nil {
			res.Streams = append(res.Streams, *streams[i])
		}
	}

	SendResponse(w, r, 200, res)
}
