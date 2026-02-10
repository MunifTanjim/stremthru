package stremio_store

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/store"
)

var stremLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:store:streamLink",
	Lifetime: 3 * time.Hour,
})

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	videoIdWithLink := r.PathValue("videoId")
	idr, err := parseId(videoIdWithLink)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idPrefix := getIdPrefix(idr.getStoreCode())
	if !strings.HasPrefix(videoIdWithLink, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "failed to get request context").Send(w, r)
		return
	}

	log := ctx.Log

	videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
	videoId, link, _ := strings.Cut(videoId, "::")

	url := link

	if url == "" {
		ctx.Log.Warn("no matching file found for (" + videoIdWithLink + ")")
		store_video.Redirect("no_matching_file", w, r)
		return
	}

	cacheKey := strings.Join([]string{ctx.ClientIP, idr.getStoreCode(), ctx.StoreAuthToken, url}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	fileName := r.PathValue("fileName")

	if idr.isUsenet {
		newzStore, ok := ctx.Store.(store.NewzStore)
		if !ok {
			log.Warn("store does not support newz")
			store_video.Redirect("500", w, r)
			return
		}
		storeName := ctx.Store.GetName()
		rParams := &store.GenerateNewzLinkParams{
			Link:     link,
			ClientIP: ctx.ClientIP,
		}
		rParams.APIKey = ctx.StoreAuthToken
		var lerr error
		data, err := newzStore.GenerateNewzLink(rParams)
		if err == nil {
			if config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) {
				if ctx.IsProxyAuthorized {
					tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
					if proxyLink, err := shared.CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, fileName); err == nil {
						data.Link = proxyLink
					} else {
						lerr = err
					}
				}
			}
		} else {
			lerr = err
		}
		if lerr != nil {
			LogError(r, "failed to generate stremthru link", lerr)
			store_video.Redirect("500", w, r)
			return
		}

		stremLinkCache.Add(cacheKey, data.Link)
		http.Redirect(w, r, data.Link, http.StatusFound)
	} else if idr.isWebDL || videoId == WEBDL_META_ID_INDICATOR {
		storeName := ctx.Store.GetName()
		rParams := &stremio_store_webdl.GenerateLinkParams{
			Link:     link,
			CLientIP: ctx.ClientIP,
		}
		rParams.APIKey = ctx.StoreAuthToken
		var lerr error
		data, err := stremio_store_webdl.GenerateLink(rParams, storeName)
		if err == nil {
			if data.Link == "" {
				store_video.Redirect(store_video.StoreVideoNameDownloading, w, r)
				return
			}
			if config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) {
				if ctx.IsProxyAuthorized {
					tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
					if proxyLink, err := shared.CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, fileName); err == nil {
						data.Link = proxyLink
					} else {
						lerr = err
					}
				}
			}
		} else {
			lerr = err
		}
		if lerr != nil {
			LogError(r, "failed to generate stremthru link", lerr)
			store_video.Redirect("500", w, r)
			return
		}

		stremLinkCache.Add(cacheKey, data.Link)
		http.Redirect(w, r, data.Link, http.StatusFound)
	} else {
		stLink, err := shared.GenerateStremThruLink(r, &ctx.Context, url, fileName)
		if err != nil {
			LogError(r, "failed to generate stremthru link", err)
			store_video.Redirect("500", w, r)
			return
		}

		stremLinkCache.Add(cacheKey, stLink.Link)
		http.Redirect(w, r, stLink.Link, http.StatusFound)
	}
}
