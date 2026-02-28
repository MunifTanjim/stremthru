package stremio_torz

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_store "github.com/MunifTanjim/stremthru/internal/stremio/store"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"golang.org/x/sync/singleflight"
)

var stremLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:wrap:streamLink",
	Lifetime: 3 * time.Hour,
})

func redirectToStaticVideo(w http.ResponseWriter, r *http.Request, cacheKey string, videoName string) {
	url := store_video.Redirect(videoName, w, r)
	stremLinkCache.AddWithLifetime(cacheKey, url, 1*time.Minute)
}

var stremGroup singleflight.Group

type stremResult struct {
	link        string
	error_level logger.Level
	error_log   string
	error_video string
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	magnetHash, encodedLink, _ := strings.Cut(r.PathValue("magnetHash"), "-")
	fileName := r.PathValue("fileName")
	fileIdx := -1
	if idx, err := strconv.Atoi(r.PathValue("fileIdx")); err == nil {
		fileIdx = idx
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

	s := ud.GetStoreByCode(r.PathValue("storeCode"))
	ctx.Store, ctx.StoreAuthToken = s.Store, s.AuthToken
	storeCode := s.Store.GetName().Code()

	cacheKey := strings.Join([]string{ctx.ClientIP, string(storeCode), ctx.StoreAuthToken, sid, magnetHash, strconv.Itoa(fileIdx), fileName}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (any, error) {
		log.Debug("creating stream link")
		amParams := &store.AddMagnetParams{
			ClientIP: ctx.ClientIP,
		}
		amParams.APIKey = ctx.StoreAuthToken
		if encodedLink == "" {
			amParams.Magnet = magnetHash
		} else {
			link, err := util.Base64Decode(encodedLink)
			if err != nil {
				return &stremResult{
					error_level: logger.LevelError,
					error_log:   "failed to decode torrent link",
					error_video: store_video.StoreVideoName500,
				}, err
			}
			magnet, fileHeader, err := shared.FetchTorrentFile(link, "", log)
			if err != nil {
				return &stremResult{
					error_level: logger.LevelError,
					error_log:   "failed to fetch torrent file",
					error_video: store_video.StoreVideoName500,
				}, err
			}
			if magnet != "" {
				amParams.Magnet = magnet
			} else {
				amParams.Torrent = fileHeader
				if _, _, err := amParams.GetTorrentMeta(); err != nil {
					return &stremResult{
						error_level: logger.LevelError,
						error_log:   "invalid torrent file",
						error_video: store_video.StoreVideoName500,
					}, err
				}
			}
		}
		amRes, err := ctx.Store.AddMagnet(amParams)
		if err != nil {
			result := &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to add magnet",
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
				case core.ErrorCodeStoreServerDown:
					result.error_level = logger.LevelWarn
					result.error_log = "store server down"
					result.error_video = store_video.StoreVideoName500
				}
			}
			return result, err
		}

		stremio_store.InvalidateCatalogCache(storeCode, ctx.StoreAuthToken)

		magnet := &store.GetMagnetData{
			Id:      amRes.Id,
			Name:    amRes.Name,
			Hash:    amRes.Hash,
			Status:  amRes.Status,
			Files:   amRes.Files,
			AddedAt: amRes.AddedAt,
		}

		isIMDBId := strings.HasPrefix(sid, "tt")
		isKitsuId := strings.HasPrefix(sid, "kitsu:")
		isMALId := strings.HasPrefix(sid, "mal:")
		isAnimeId := isKitsuId || isMALId
		shouldTagStream := isIMDBId || isAnimeId

		magnet, err = stremio_shared.WaitForMagnetStatus(&ctx.Ctx, magnet, store.MagnetStatusDownloaded, 3, 5*time.Second)
		if err != nil {
			strem := &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed wait for magnet status",
				error_video: store_video.StoreVideoName500,
			}
			switch magnet.Status {
			case store.MagnetStatusQueued, store.MagnetStatusDownloading, store.MagnetStatusProcessing:
				strem.error_level = logger.LevelWarn
				strem.error_video = store_video.StoreVideoNameDownloading
			case store.MagnetStatusFailed, store.MagnetStatusInvalid, store.MagnetStatusUnknown:
				strem.error_level = logger.LevelWarn
				strem.error_video = store_video.StoreVideoNameDownloadFailed
			}
			return strem, err
		}

		go buddy.TrackMagnet(ctx.Store, magnet.Hash, magnet.Name, magnet.Size, magnet.Private, magnet.Files, torrent_info.GetCategoryFromStremId(sid, ""), magnet.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)

		videoFiles := []store.File{}
		for i := range magnet.Files {
			f := &magnet.Files[i]
			if core.HasVideoExtension(f.Name) {
				videoFiles = append(videoFiles, f)
			}
		}

		var file store.File
		if strings.Contains(sid, ":") {
			if file = stremio_shared.MatchFileByStremId(magnet.Name, videoFiles, sid, magnetHash, storeCode); file != nil {
				log.Debug("matched file using strem id", "sid", sid, "filename", file.GetName())
			}
		}
		if file == nil && fileName != "" {
			if file = stremio_shared.MatchFileByName(videoFiles, fileName); file != nil {
				log.Debug("matched file using filename", "filename", file.GetName())
			}
		}
		if file == nil {
			if file = stremio_shared.MatchFileByIdx(videoFiles, fileIdx, storeCode); file != nil {
				log.Debug("matched file using fileidx", "fileidx", file.GetIdx(), "filename", file.GetName())
			}
		}
		if file == nil && isIMDBId && (!strings.Contains(sid, ":") || len(videoFiles) == 1) {
			if file = stremio_shared.MatchFileByLargestSize(videoFiles); file != nil {
				log.Debug("matched file using largest size", "filename", file.GetName())
				shouldTagStream = len(videoFiles) == 1
			}
		}

		link := ""
		if file != nil {
			link = file.GetLink()
		}
		if link == "" {
			return &stremResult{
				error_level: logger.LevelWarn,
				error_log:   "no matching file found for (" + sid + " - " + magnet.Hash + ")",
				error_video: store_video.StoreVideoNameNoMatchingFile,
			}, nil
		}

		if shouldTagStream {
			if isIMDBId {
				torrent_stream.TagStremId(magnet.Hash, file.GetPath(), sid)
			} else if isAnimeId {
				go torrent_stream.TagAnimeStremId(magnet.Hash, file.GetPath(), sid)
			}
		}

		glRes, err := shared.GenerateStremThruLink(r, &ctx.Context, link, fileName)
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to generate stremthru link",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		stremLinkCache.Add(cacheKey, glRes.Link)

		if storeCode == store.StoreCodeTorBox {
			torrent_stream.QueueMediaInfoProbe(magnet.Hash, file.GetPath(), glRes.Link)
		}

		return &stremResult{
			link: glRes.Link,
		}, nil
	})

	strem := result.(*stremResult)

	if strem.error_log != "" {
		log.Log(strem.error_level, strem.error_log, "error", err, "store.name", ctx.Store.GetName())
		redirectToStaticVideo(w, r, cacheKey, strem.error_video)
		return
	}

	log.Debug("redirecting to stream link")
	http.Redirect(w, r, strem.link, http.StatusFound)
}
