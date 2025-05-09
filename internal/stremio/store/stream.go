package stremio_store

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

var streamTemplate = func() *stremio_transformer.StreamTemplate {
	tmplBlob := stremio_transformer.StreamTemplateBlob{
		Name: `Store {{.Store.Code}}
{{ if ne .Resolution ""}}{{.Resolution}}{{end}}`,
		Description: `✏️ {{.TTitle}}
{{if ne .Quality ""}} 💿 {{.Quality}} {{end}}{{if ne .Codec ""}} 🎞️ {{.Codec}} {{end}}{{if gt (len .HDR) 0}} 📺 {{str_join .HDR ","}}{{end}}{{if gt (len .Audio) 0}} 🎧 {{str_join .Audio ","}}{{if gt (len .Channels) 0}} | {{str_join .Channels ","}}{{end}}{{end}}
{{if ne .Size ""}} 📦 {{.Size}}{{end}}{{if ne .Group ""}} ⚙️ {{.Group}}{{end}}
📄 {{.Raw.Name}}`,
	}
	tmpl, err := tmplBlob.Parse()
	if err != nil {
		panic(err)
	}
	return tmpl
}()

type StreamFileMatcher struct {
	MagnetId       string
	FileLink       string
	FileName       string
	UseLargestFile bool
	Episode        int
	Season         int

	IdPrefix   string
	Store      store.Store
	StoreCode  string
	StoreToken string
	ClientIP   string
	IsUsenet   bool
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
			Store:      ctx.Store,
			StoreCode:  idr.getStoreCode(),
			StoreToken: ctx.StoreAuthToken,
			ClientIP:   ctx.ClientIP,
			IsUsenet:   idr.isUsenet,
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

				isUsenet := strings.HasSuffix(idPrefix, "-usenet:")
				items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, idPrefix, isUsenet)
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
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
							IsUsenet:   isUsenet,
						})
					} else {
						matcherResults[idx] = append(matcherResults[idx], StreamFileMatcher{
							MagnetId:       id,
							UseLargestFile: true,

							IdPrefix:   idPrefix,
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
							IsUsenet:   isUsenet,
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

			cInfo, err := getStoreContentInfo(matcher.Store, matcher.StoreToken, matcher.MagnetId, matcher.ClientIP, matcher.IsUsenet)
			if err != nil {
				errs[i] = err
				return
			}

			if !matcher.IsUsenet && meta == nil {
				stremIdByHash, err := torrent_stream.GetStremIdByHashes([]string{cInfo.Hash})
				if err != nil {
					log.Error("failed to get strem id by hashes", "error", err)
				}
				if stremId := stremIdByHash.Get(cInfo.Hash); stremId != "" {
					sType, sId := "", ""
					sType, sId, season, episode = parseStremId(stremId)
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
			tSeason := -1
			if len(tpttr.Seasons) == 1 {
				tSeason = tpttr.Seasons[0]
			}

			var pttr *ptt.Result
			var file *store.MagnetFile

			for i := range cInfo.Files {
				f := &cInfo.Files[i]
				if matcher.FileLink != "" && matcher.FileLink == f.Link {
					file = f
					break
				} else if matcher.FileName != "" && matcher.FileName == f.Name {
					file = f
					break
				} else if matcher.Episode > 0 {
					if r, err := util.ParseTorrentTitle(f.Name); err == nil {
						pttr = r
						season, episode := tSeason, -1
						if len(r.Seasons) > 0 {
							season = r.Seasons[0]
						}
						if len(r.Episodes) > 0 {
							episode = r.Episodes[0]
						}
						if season == matcher.Season && episode == matcher.Episode {
							file = f
							break
						}
					} else {
						pttLog.Warn("failed to parse", "error", err, "title", f.Name)
					}
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
						if video.Season == season && video.Episode == episode {
							pttr.Title = video.Name
							break
						}
					}
				}
				data := &stremio_transformer.StreamExtractorResult{
					Result: pttr,
					Raw: stremio_transformer.StreamExtractorResultRaw{
						Name:        stream.Name,
						Description: stream.Description,
					},
					Store: stremio_transformer.StreamExtractorResultStore{
						Code:     strings.ToUpper(matcher.StoreCode),
						Name:     string(store.StoreCode(matcher.StoreCode).Name()),
						IsCached: true,
					},
				}
				if stream.Description == "" {
					data.Raw.Description = stream.Title
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
