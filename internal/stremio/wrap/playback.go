package stremio_wrap

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
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
	error_log   string
	error_video string
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	magnetHash := r.PathValue("magnetHash")
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

	query := r.URL.Query()

	s := ud.GetStoreByCode(query.Get("s"))
	ctx.Store, ctx.StoreAuthToken = s.Store, s.AuthToken
	storeCode := ctx.Store.GetName().Code()

	cacheKey := strings.Join([]string{ctx.ClientIP, string(storeCode), ctx.StoreAuthToken, magnetHash, strconv.Itoa(fileIdx), fileName, query.Encode()}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (any, error) {
		log.Debug("creating stream link")
		amParams := &store.AddMagnetParams{
			Magnet:   magnetHash,
			ClientIP: ctx.ClientIP,
		}
		amParams.APIKey = ctx.StoreAuthToken
		amRes, err := ctx.Store.AddMagnet(amParams)
		if err != nil {
			return &stremResult{
				error_log:   "failed to add magnet",
				error_video: "download_failed",
			}, err
		}

		magnet := &store.GetMagnetData{
			Id:      amRes.Id,
			Name:    amRes.Name,
			Hash:    amRes.Hash,
			Status:  amRes.Status,
			Files:   amRes.Files,
			AddedAt: amRes.AddedAt,
		}

		magnet, err = stremio_shared.WaitForMagnetStatus(ctx, magnet, store.MagnetStatusDownloaded, 3, 5*time.Second)
		if err != nil {
			strem := &stremResult{
				error_log:   "failed wait for magnet status",
				error_video: "500",
			}
			if magnet.Status == store.MagnetStatusQueued || magnet.Status == store.MagnetStatusDownloading || magnet.Status == store.MagnetStatusProcessing {
				strem.error_video = "downloading"
			} else if magnet.Status == store.MagnetStatusFailed || magnet.Status == store.MagnetStatusInvalid || magnet.Status == store.MagnetStatusUnknown {
				strem.error_video = "download_failed"
			}
			return strem, err
		}

		sid := query.Get("sid")
		if sid == "" {
			sid = "*"
		}

		go buddy.TrackMagnet(ctx.Store, magnet.Hash, magnet.Name, magnet.Size, magnet.Files, torrent_info.GetCategoryFromStremId(sid), magnet.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)

		var pattern *regexp.Regexp
		if re := query.Get("re"); re != "" {
			if pat, err := regexp.Compile(re); err == nil {
				pattern = pat
			}
		}

		shouldTagStream := strings.HasPrefix(sid, "tt")

		videoFiles := []store.MagnetFile{}
		for i := range magnet.Files {
			f := &magnet.Files[i]
			if core.HasVideoExtension(f.Name) {
				videoFiles = append(videoFiles, *f)
			}
		}

		var file *store.MagnetFile
		if fileName != "" {
			if file = stremio_shared.MatchFileByName(videoFiles, fileName); file != nil {
				log.Debug("matched file using filename", "filename", file.Name)
			}
		}
		if file == nil && strings.Contains(sid, ":") {
			if file = stremio_shared.MatchFileByStremId(videoFiles, sid, magnetHash, storeCode); file != nil {
				log.Debug("matched file using stream id", "sid", sid, "filename", file.Name)
			}
		}
		if file == nil {
			if file = stremio_shared.MatchFileByIdx(videoFiles, fileIdx, storeCode); file != nil {
				log.Debug("matched file using fileidx", "fileidx", file.Idx, "filename", file.Name)
			}
		}
		if file == nil && pattern != nil {
			if file = stremio_shared.MatchFileByPattern(videoFiles, pattern); file != nil {
				log.Debug("matched file using pattern", "pattern", pattern.String(), "filename", file.Name)
			}
		}
		if file == nil {
			if file = stremio_shared.MatchFileByLargestSize(videoFiles); file != nil {
				log.Debug("matched file using largest size", "filename", file.Name)
				shouldTagStream = false
			}
		}

		link := ""
		if file != nil {
			link = file.Link
		}
		if link == "" {
			return &stremResult{
				error_log:   "no matching file found for (" + sid + " - " + magnet.Hash + ")",
				error_video: "no_matching_file",
			}, nil
		}

		if shouldTagStream {
			torrent_stream.TagStremId(magnet.Hash, file.Name, sid)
		}

		glRes, err := shared.GenerateStremThruLink(r, ctx, link)
		if err != nil {
			return &stremResult{
				error_log:   "failed to generate stremthru link",
				error_video: "500",
			}, err
		}

		stremLinkCache.Add(cacheKey, glRes.Link)

		return &stremResult{
			link: glRes.Link,
		}, nil
	})

	strem := result.(*stremResult)

	if strem.error_log != "" {
		if err != nil {
			LogError(r, strem.error_log, err)
		} else {
			log.Error(strem.error_log)
		}
		redirectToStaticVideo(w, r, cacheKey, strem.error_video)
		return
	}

	log.Debug("redirecting to stream link")
	http.Redirect(w, r, strem.link, http.StatusFound)
}
