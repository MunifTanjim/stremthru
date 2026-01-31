package stremio_newz

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/logger"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_store "github.com/MunifTanjim/stremthru/internal/stremio/store"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/store"
	"golang.org/x/sync/singleflight"
)

var stremLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:newz:streamLink",
	Lifetime: 3 * time.Hour,
})

func redirectToStaticVideo(w http.ResponseWriter, r *http.Request, cacheKey string, videoName string) {
	url := store_video.Redirect(videoName, w, r)
	if cacheKey != "" {
		stremLinkCache.AddWithLifetime(cacheKey, url, 1*time.Minute)
	}
}

var stremGroup singleflight.Group

type stremResult struct {
	link        string
	error_level logger.Level
	error_log   string
	error_video string
}

func handleStremFromStore(w http.ResponseWriter, r *http.Request, ud *UserData, ctx *RequestContext, sid string, storeCode store.StoreCode, nzbUrl string) {
	log := server.GetReqCtx(r).Log

	s := ud.GetStoreByCode(string(storeCode))
	if s.Store == nil {
		log.Warn("store not found", "store.code", storeCode)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	newzStore, ok := s.Store.(store.NewzStore)
	if !ok {
		log.Warn("store does not support newz", "store.name", s.Store.GetName())
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	ctx.Store, ctx.StoreAuthToken = s.Store, s.AuthToken

	cacheKey := strings.Join([]string{ctx.ClientIP, string(storeCode), ctx.StoreAuthToken, sid, nzbUrl}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (any, error) {
		addParams := &store.AddNewzParams{
			Link: nzbUrl,
		}
		addParams.APIKey = ctx.StoreAuthToken
		addRes, err := newzStore.AddNewz(addParams)
		if err != nil {
			result := &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to add newz",
				error_video: store_video.StoreVideoNameDownloadFailed,
			}
			var uerr *core.UpstreamError
			if errors.As(err, &uerr) {
				switch uerr.Code {
				case core.ErrorCodeUnauthorized:
					result.error_level = logger.LevelWarn
					result.error_log = "unauthorized"
					result.error_video = store_video.StoreVideoName401
				case core.ErrorCodeTooManyRequests:
					result.error_level = logger.LevelWarn
					result.error_log = "too many requests"
					result.error_video = store_video.StoreVideoName429
				case core.ErrorCodeUnavailableForLegalReasons:
					result.error_level = logger.LevelWarn
					result.error_log = "unavaiable for legal reason"
					result.error_video = store_video.StoreVideoName451
				case core.ErrorCodePaymentRequired:
					result.error_level = logger.LevelWarn
					result.error_log = "payment required"
					result.error_video = store_video.StoreVideoNamePaymentRequired
				case core.ErrorCodeStoreLimitExceeded:
					result.error_log = "store limit exceeded"
					result.error_video = store_video.StoreVideoNameStoreLimitExceeded
				}
			}
			return result, err
		}

		stremio_store.InvalidateCatalogCache(storeCode, ctx.StoreAuthToken)

		getParams := &store.GetNewzParams{
			Id:       addRes.Id,
			ClientIP: ctx.ClientIP,
		}
		getParams.APIKey = ctx.StoreAuthToken
		newz, err := newzStore.GetNewz(getParams)
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to get newz",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		if newz.Status != store.NewzStatusDownloaded {
			strem := &stremResult{
				error_level: logger.LevelError,
				error_log:   "newz not downloaded",
				error_video: store_video.StoreVideoName500,
			}
			switch newz.Status {
			case store.NewzStatusQueued, store.NewzStatusDownloading, store.NewzStatusProcessing:
				strem.error_level = logger.LevelWarn
				strem.error_video = store_video.StoreVideoNameDownloading
			case store.NewzStatusFailed, store.NewzStatusInvalid, store.NewzStatusUnknown:
				strem.error_level = logger.LevelWarn
				strem.error_video = store_video.StoreVideoNameDownloadFailed
			}
			return strem, nil
		}

		videoFiles := []store.File{}
		for i := range newz.Files {
			f := &newz.Files[i]
			if core.HasVideoExtension(f.Name) {
				videoFiles = append(videoFiles, f)
			}
		}

		var file store.File
		isIMDBId := strings.HasPrefix(sid, "tt")

		if strings.Contains(sid, ":") {
			if file = stremio_shared.MatchFileByStremId(newz.Name, videoFiles, sid, "", storeCode); file != nil {
				log.Debug("matched file using strem id", "sid", sid, "filename", file.GetName())
			}
		}
		if file == nil && isIMDBId && (!strings.Contains(sid, ":") || len(videoFiles) == 1) {
			if file = stremio_shared.MatchFileByLargestSize(videoFiles); file != nil {
				log.Debug("matched file using largest size", "filename", file.GetName())
			}
		}

		link := ""
		if file != nil {
			link = file.GetLink()
		}
		if link == "" {
			return &stremResult{
				error_level: logger.LevelWarn,
				error_log:   "no matching file found for (" + sid + " - " + newz.Hash + ")",
				error_video: store_video.StoreVideoNameNoMatchingFile,
			}, nil
		}

		linkParams := &store.GenerateNewzLinkParams{
			Link:     file.GetLink(),
			ClientIP: ctx.ClientIP,
		}
		linkParams.APIKey = ctx.StoreAuthToken
		linkRes, err := newzStore.GenerateNewzLink(linkParams)
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to generate download link",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		link, err = shared.ProxyWrapLink(r, ctx.StoreContext, linkRes.Link, file.GetName())
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to generate stremthru link",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		stremLinkCache.Add(cacheKey, link)

		return &stremResult{
			link: link,
		}, nil
	})

	strem := result.(*stremResult)

	if strem.error_log != "" {
		log.Log(strem.error_level, strem.error_log, "error", err)
		redirectToStaticVideo(w, r, cacheKey, strem.error_video)
		return
	}

	log.Debug("redirecting to stream link")
	http.Redirect(w, r, strem.link, http.StatusFound)
}

func handleStreamFromUsenet(w http.ResponseWriter, r *http.Request, nzbUrl string) {
	ctx := context.Background()

	log.Debug("starting direct NNTP stream")

	pool, err := usenetmanager.GetPool(log)
	if err != nil {
		log.Error("failed to get global NNTP pool", "error", err)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}
	if pool == nil {
		log.Warn("no NNTP providers configured")
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	newz := newznab_client.Newz{DownloadLink: nzbUrl}
	nzbDoc, err := newz.FetchNZB(log)
	if err != nil {
		redirectToStaticVideo(w, r, "", store_video.StoreVideoNameDownloadFailed)
		return
	}

	// fileIdx := findVideoFileIdx(nzbDoc, sid, log)
	// if fileIdx < 0 {
	// 	log.Warn("no video files found in NZB")
	// 	redirectToStaticVideo(w, r, "", store_video.StoreVideoNameNoMatchingFile)
	// 	return
	// }

	stream, err := pool.StreamLargestFile(ctx, nzbDoc, nil)
	if err != nil {
		log.Error("failed to create usenet stream", "error", err)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, stream.Name, time.Now(), stream)
	return
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	encodedNzbUrl, err := url.PathUnescape(r.PathValue("nzbUrl"))
	if err != nil {
		shared.ErrorBadRequest(r, "invalid nzbUrl").Send(w, r)
		return
	}
	nzbUrl, err := core.Base64Decode(encodedNzbUrl)
	if err != nil {
		shared.ErrorBadRequest(r, "invalid nzbUrl encoding").Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		LogError(r, "failed to get request context", err)
		shared.ErrorBadRequest(r, "failed to get request context: "+err.Error()).Send(w, r)
		return
	}

	sid := r.PathValue("stremId")
	storeCode := store.StoreCode(r.PathValue("storeCode"))

	if stremio_store_usenet.IsSupported(storeCode) {
		handleStremFromStore(w, r, ud, ctx, sid, storeCode, nzbUrl)
		return
	}

	isStremThruStore := storeCode == "st"
	if !isStremThruStore {
		log.Warn("non-stremthru store not supported for NZB streaming", "storeCode", storeCode)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	handleStreamFromUsenet(w, r, nzbUrl)
	return
}
