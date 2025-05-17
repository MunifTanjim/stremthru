package stremio_store

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
)

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

	videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
	videoId, link, _ := strings.Cut(videoId, ":")

	url := link

	if url == "" {
		ctx.Log.Warn("no matching file found for (" + videoIdWithLink + ")")
		store_video.Redirect("no_matching_file", w, r)
		return
	}

	if idr.isUsenet {
		storeName := ctx.Store.GetName()
		rParams := &stremio_store_usenet.GenerateLinkParams{
			Link:     link,
			CLientIP: ctx.ClientIP,
		}
		rParams.APIKey = ctx.StoreAuthToken
		var lerr error
		data, err := stremio_store_usenet.GenerateLink(rParams, storeName)
		if err == nil {
			if config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) {
				if ctx.IsProxyAuthorized {
					tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
					if proxyLink, err := shared.CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, ""); err == nil {
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

		http.Redirect(w, r, data.Link, http.StatusFound)
	} else if idr.isWebDL {
		storeName := ctx.Store.GetName()
		rParams := &stremio_store_webdl.GenerateLinkParams{
			Link:     link,
			CLientIP: ctx.ClientIP,
		}
		rParams.APIKey = ctx.StoreAuthToken
		var lerr error
		data, err := stremio_store_webdl.GenerateLink(rParams, storeName)
		if err == nil {
			if config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) {
				if ctx.IsProxyAuthorized {
					tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
					if proxyLink, err := shared.CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, ""); err == nil {
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

		http.Redirect(w, r, data.Link, http.StatusFound)
	} else {
		stLink, err := shared.GenerateStremThruLink(r, ctx, url)
		if err != nil {
			LogError(r, "failed to generate stremthru link", err)
			store_video.Redirect("500", w, r)
			return
		}

		http.Redirect(w, r, stLink.Link, http.StatusFound)
	}
}
