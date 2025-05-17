package stremio_store

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	"github.com/MunifTanjim/stremthru/store"
)

func handleAction(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	actionId := r.PathValue("actionId")
	idr, err := parseId(actionId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	storeActionIdPrefix := getStoreActionIdPrefix(idr.getStoreCode())
	if !strings.HasPrefix(actionId, storeActionIdPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+actionId).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		store_video.Redirect("500", w, r)
		return
	}

	idPrefix := getIdPrefix(idr.getStoreCode())
	switch strings.TrimPrefix(actionId, storeActionIdPrefix) {
	case "clear_cache":
		catalogCache.Remove(getCatalogCacheKey(idPrefix, ctx.StoreAuthToken))
		if ctx.Store.GetName() == store.StoreNameRealDebrid {
			rdDownloadsCache.Remove(getRDDownloadsCacheKey(ctx.StoreAuthToken))
		}
	}

	store_video.Redirect("200", w, r)
}
