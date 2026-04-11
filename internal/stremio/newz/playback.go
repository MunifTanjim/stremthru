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
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"golang.org/x/sync/singleflight"
)

var streamLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:newz:stream-link",
	Lifetime: 3 * time.Hour,
})

func redirectToStaticVideo(w http.ResponseWriter, r *http.Request, cacheKey string, videoName string) {
	url := store_video.Redirect(videoName, w, r)
	if cacheKey != "" {
		streamLinkCache.AddWithLifetime(cacheKey, url, 1*time.Minute)
	}
}

var stremGroup singleflight.Group

type stremResult struct {
	link        string
	error_level logger.Level
	error_log   string
	error_video string
}

var usenetStremGroup singleflight.Group

type usenetStremResult struct {
	hash         string
	contentPath  string
	streamConfig *usenet_pool.StreamConfig
	nzbDoc       *nzb.NZB
	nzbFileMod   time.Time
	error_level  logger.Level
	error_log    string
	error_video  string
}

func handlePlaybackFromStore(w http.ResponseWriter, r *http.Request, ud *UserData, ctx *Ctx, sid string, storeCode store.StoreCode, nzbUrl string, isLockedDownload bool) {
	log := server.GetReqCtx(r).Log

	s := ud.GetStoreByCode(string(storeCode))
	if s.Store == nil {
		log.Warn("store not found", "store.code", storeCode)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	newzStore, ok := s.Store.(store.NewzStore)
	if !ok {
		shared.ErrorBadRequest(r, "unsupported store for newz playback").Send(w, r)
		return
	}

	ctx.Store, ctx.StoreAuthToken = s.Store, s.AuthToken

	cacheKey := strings.Join([]string{ctx.ClientIP, string(storeCode), ctx.StoreAuthToken, sid, nzbUrl}, ":")

	stremLink := ""
	if streamLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (any, error) {
		addParams := &store.AddNewzParams{
			ClientIP: ctx.ClientIP,
		}
		if isLockedDownload {
			addParams.Link = nzbUrl
		} else if u, err := url.Parse(nzbUrl); err == nil {
			hostname := u.Hostname()
			if config.NewzNZBLinkMode.Redirect(hostname) {
				addParams.Link = nzbUrl
			} else if config.NewzNZBLinkMode.Proxy(hostname) {
				nzbFile, err := nzb_info.FetchNZBFile(nzbUrl, r.PathValue("fileName"), log)
				if err != nil {
					return &stremResult{
						error_level: logger.LevelError,
						error_log:   "failed to fetch nzb",
						error_video: store_video.StoreVideoNameDownloadFailed,
					}, err
				}
				fileHeader, err := nzbFile.ToFileHeader()
				if err != nil {
					return &stremResult{
						error_level: logger.LevelError,
						error_log:   "failed to prepare nzb",
						error_video: store_video.StoreVideoName500,
					}, err
				}
				addParams.File = fileHeader
			} else {
				return &stremResult{
					error_level: logger.LevelWarn,
					error_log:   "nzb host not allowed",
					error_video: store_video.StoreVideoNameDownloadFailed,
				}, nil
			}
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

		retryInterval := 5 * time.Second
		maxRetry := int(config.Stremio.Newz.PlaybackWaitTime / retryInterval)
		newz, err := stremio_shared.WaitForNewzStatus(&ctx.Ctx, &store.GetNewzData{
			Id:     addRes.Id,
			Hash:   addRes.Hash,
			Status: addRes.Status,
		}, store.NewzStatusDownloaded, maxRetry, retryInterval)
		if err != nil {
			strem := &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed wait for newz status",
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
			return strem, err
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

		link, err = shared.ProxyWrapLink(r, &ctx.Context, linkRes.Link, file.GetName())
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to generate stremthru link",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		streamLinkCache.Add(cacheKey, link)

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

func handleStreamFromUsenet(w http.ResponseWriter, r *http.Request, ud *UserData, ctx *Ctx, sid string, nzbUrl string) {
	log := server.GetReqCtx(r).Log

	pool, err := usenetmanager.GetPool()
	if err != nil {
		log.Error("failed to get NNTP pool", "error", err)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}
	if pool == nil {
		log.Warn("no NNTP providers configured")
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	ctx.Store = shared.GetStore(string(store.StoreNameStremThru))
	if ctx.Store == nil {
		log.Warn("store not found", "store.name", store.StoreNameStremThru)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}
	ctx.StoreAuthToken = ctx.ProxyAuthUser + ":" + ctx.ProxyAuthPassword

	newzStore, ok := ctx.Store.(store.NewzStore)
	if !ok {
		log.Warn("unsupported store for newz playback")
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}

	cacheKey := nzb_info.HashNZBFileLink(nzbUrl)
	fileName := r.PathValue("fileName")

	result, err, _ := usenetStremGroup.Do(cacheKey, func() (any, error) {
		addParams := &store.AddNewzParams{
			Link:     nzbUrl,
			ClientIP: ctx.ClientIP,
		}
		addParams.APIKey = ctx.StoreAuthToken
		addRes, err := newzStore.AddNewz(addParams)
		if err != nil {
			return &usenetStremResult{
				error_level: logger.LevelError,
				error_log:   "failed to add newz",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		retryInterval := 5 * time.Second
		maxRetry := int(config.Stremio.Newz.PlaybackWaitTime / retryInterval)
		newz, err := stremio_shared.WaitForNewzStatus(&ctx.Ctx, &store.GetNewzData{
			Id:     addRes.Id,
			Hash:   addRes.Hash,
			Status: addRes.Status,
		}, store.NewzStatusDownloaded, maxRetry, retryInterval)
		if err != nil {
			strem := &usenetStremResult{
				error_level: logger.LevelError,
				error_log:   "failed to wait for newz status",
				error_video: store_video.StoreVideoName500,
			}
			switch newz.Status {
			case store.NewzStatusQueued, store.NewzStatusDownloading, store.NewzStatusProcessing:
				strem.error_level = logger.LevelWarn
				strem.error_log = "newz not ready"
				strem.error_video = store_video.StoreVideoNameDownloading
			case store.NewzStatusFailed, store.NewzStatusInvalid, store.NewzStatusUnknown:
				strem.error_level = logger.LevelWarn
				strem.error_log = "newz failed"
				strem.error_video = store_video.StoreVideoNameDownloadFailed
			}
			return strem, err
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
			if file = stremio_shared.MatchFileByStremId(newz.Name, videoFiles, sid, "", ctx.Store.GetName().Code()); file != nil {
				log.Debug("matched file using strem id", "sid", sid, "filename", file.GetName())
			}
		}
		if file == nil && isIMDBId && (!strings.Contains(sid, ":") || len(videoFiles) == 1) {
			if file = stremio_shared.MatchFileByLargestSize(videoFiles); file != nil {
				log.Debug("matched file using largest size", "filename", file.GetName())
			}
		}

		if file == nil {
			return &usenetStremResult{
				error_level: logger.LevelWarn,
				error_log:   "no matching file found for (" + sid + " - " + newz.Hash + ")",
				error_video: store_video.StoreVideoNameNoMatchingFile,
			}, nil
		}

		hash := newz.Hash
		info, err := nzb_info.GetByHash(hash)
		if err != nil {
			return &usenetStremResult{
				error_level: logger.LevelError,
				error_log:   "failed to get nzb info",
				error_video: store_video.StoreVideoName500,
			}, err
		}
		if info == nil || !info.Streamable {
			return &usenetStremResult{
				error_level: logger.LevelWarn,
				error_log:   "nzb is not streamable",
				error_video: store_video.StoreVideoNameDownloadFailed,
			}, nil
		}

		nzbFile, err := nzb_info.FetchNZBFile(nzbUrl, fileName, log)
		if err != nil {
			return &usenetStremResult{
				error_level: logger.LevelError,
				error_log:   "failed to fetch nzb",
				error_video: store_video.StoreVideoNameDownloadFailed,
			}, err
		}
		nzbDoc, err := nzb.ParseBytes(nzbFile.Blob)
		if err != nil {
			return &usenetStremResult{
				error_level: logger.LevelError,
				error_log:   "failed to parse nzb",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		return &usenetStremResult{
			hash:        hash,
			contentPath: file.GetPath(),
			streamConfig: &usenet_pool.StreamConfig{
				Password:     info.Password,
				ContentFiles: info.ContentFiles.Data,
			},
			nzbDoc:     nzbDoc,
			nzbFileMod: nzbFile.Mod,
		}, nil
	})

	strem := result.(*usenetStremResult)

	if strem.error_log != "" {
		log.Log(strem.error_level, strem.error_log, "error", err)
		redirectToStaticVideo(w, r, "", strem.error_video)
		return
	}

	streamCtx := context.WithValue(r.Context(), usenet_pool.NZBHashContextKey, strem.hash)
	stream, err := pool.StreamByContentPath(streamCtx, strem.nzbDoc, strem.contentPath, strem.streamConfig)
	if err != nil {
		log.Error("failed to create usenet stream", "error", err)
		redirectToStaticVideo(w, r, "", store_video.StoreVideoName500)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, stream.Name, strem.nzbFileMod, stream)
}

func handlePlayback(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	encodedNzbUrl, err := url.PathUnescape(r.PathValue("nzbUrl"))
	if err != nil {
		shared.ErrorBadRequest(r, "invalid nzbUrl").Send(w, r)
		return
	}
	var isLockedDownload bool
	encodedNzbUrl, isLockedDownload = strings.CutPrefix(encodedNzbUrl, "-")
	nzbUrl, err := util.Base64Decode(encodedNzbUrl)
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

	switch UserDataMode(r.PathValue("mode")) {
	case UserDataModeDebrid:
		storeCode := store.StoreCode(r.PathValue("storeCode"))

		if !stremio_store_usenet.IsSupported(storeCode) {
			shared.ErrorBadRequest(r, "unsupported store for newz playback").Send(w, r)
			return
		}

		handlePlaybackFromStore(w, r, ud, ctx, sid, storeCode, nzbUrl, isLockedDownload)
	case UserDataModeStream:
		if !ctx.IsProxyAuthorized {
			redirectToStaticVideo(w, r, "", store_video.StoreVideoName403)
			return
		}
		handleStreamFromUsenet(w, r, ud, ctx, sid, nzbUrl)
	default:
		shared.ErrorBadRequest(r, "invalid mode").Send(w, r)
	}
}
