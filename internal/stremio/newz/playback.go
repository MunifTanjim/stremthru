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
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"golang.org/x/sync/singleflight"
)

var stremLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:newz:streamLink",
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

// findVideoFileIdx finds the index of the appropriate video file in the NZB
func findVideoFileIdx(nzbDoc *nzb.NZB, sid string, log *logger.Logger) int {
	type videoFile struct {
		idx  int
		name string
		size int64
	}
	var videoFiles []videoFile

	for i := range nzbDoc.Files {
		f := &nzbDoc.Files[i]
		name := f.GetName()
		if core.HasVideoExtension(name) {
			videoFiles = append(videoFiles, videoFile{
				idx:  i,
				name: name,
				size: f.TotalSize(),
			})
		}
	}

	if len(videoFiles) == 0 {
		return -1
	}

	// For series with episode info, try to match by filename
	if strings.Contains(sid, ":") && len(videoFiles) > 1 {
		parts := strings.Split(sid, ":")
		if len(parts) >= 3 {
			season := util.SafeParseInt(parts[1], -1)
			episode := util.SafeParseInt(parts[2], -1)
			if season > 0 && episode > 0 {
				for i := range videoFiles {
					vf := &videoFiles[i]
					pttr, err := util.ParseTorrentTitle(vf.name)
					if err != nil {
						continue
					}
					// Check if this file matches the requested season/episode
					seasonMatch := len(pttr.Seasons) == 0 || (len(pttr.Seasons) > 0 && pttr.Seasons[0] == season)
					episodeMatch := len(pttr.Episodes) == 0 || (len(pttr.Episodes) > 0 && pttr.Episodes[0] == episode)
					if seasonMatch && episodeMatch && len(pttr.Episodes) > 0 {
						log.Debug("matched video file by episode", "filename", vf.name, "season", season, "episode", episode)
						return vf.idx
					}
				}
			}
		}
	}

	// Default: return largest video file
	largestIdx := 0
	for i := range videoFiles {
		if videoFiles[i].size > videoFiles[largestIdx].size {
			largestIdx = i
		}
	}
	log.Debug("using largest video file", "filename", videoFiles[largestIdx].name)
	return videoFiles[largestIdx].idx
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

	// Direct NNTP streaming - handle without singleflight to support range requests
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

	if err := streamNZBContent(w, r, pool, nzbDoc, -1); err != nil {
		log.Error("streaming failed", "error", err)
	}
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

	storeAuthToken := s.AuthToken

	cacheKey := strings.Join([]string{ctx.ClientIP, string(storeCode), storeAuthToken, sid, nzbUrl}, ":")

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
		addParams.APIKey = storeAuthToken
		addRes, err := newzStore.AddNewz(addParams)
		if err != nil {
			result := &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to add NZB to store",
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

		getParams := &store.GetNewzParams{
			Id:       addRes.Id,
			ClientIP: ctx.ClientIP,
		}
		getParams.APIKey = storeAuthToken
		newz, err := newzStore.GetNewz(getParams)
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to get NZB download details",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		if newz.Status != store.NewzStatusDownloaded {
			return &stremResult{
				error_level: logger.LevelWarn,
				error_log:   "NZB not cached/downloaded yet",
				error_video: store_video.StoreVideoNameDownloading,
			}, nil
		}

		videoFiles := []store.File{}
		for i := range newz.Files {
			f := &newz.Files[i]
			if core.HasVideoExtension(f.Name) {
				videoFiles = append(videoFiles, f)
			}
		}

		if len(videoFiles) == 0 {
			return &stremResult{
				error_level: logger.LevelWarn,
				error_log:   "no video files found in NZB",
				error_video: store_video.StoreVideoNameNoMatchingFile,
			}, nil
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
		linkParams.APIKey = storeAuthToken
		linkRes, err := newzStore.GenerateNewzLink(linkParams)
		if err != nil {
			return &stremResult{
				error_level: logger.LevelError,
				error_log:   "failed to generate download link",
				error_video: store_video.StoreVideoName500,
			}, err
		}

		link, err = shared.ProxyWrapLink(r, ctx.StoreContext, linkRes.Link)
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

// streamNZBContent streams content from an NZB file via NNTP
func streamNZBContent(w http.ResponseWriter, r *http.Request, pool *usenet_pool.Pool, nzbDoc *nzb.NZB, fileIdx int) error {
	ctx := context.Background()

	var stream *usenet_pool.Stream
	var err error

	if fileIdx >= 0 && fileIdx < nzbDoc.FileCount() {
		stream, err = pool.StreamFileByName(ctx, nzbDoc, nzbDoc.Files[fileIdx].GetName(), nil)
	} else {
		stream, err = pool.StreamLargestFile(ctx, nzbDoc, nil)
	}

	if err != nil {
		return err
	}
	defer stream.Close()

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, stream.Name, time.Now(), stream)
	return nil
}
