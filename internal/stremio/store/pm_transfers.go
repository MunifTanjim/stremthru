package stremio_store

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/stremio"
)

var pmItemsCache = cache.NewCache[[]stremio.MetaVideo](&cache.CacheConfig{
	Lifetime: 1 * time.Minute,
	Name:     "stremio:store:pm:items",
})

func getPMItemsCacheKey(idStoreCode, storeToken string) string {
	return getCatalogCacheKey(idStoreCode, storeToken)
}

func getPMWebDLsMeta(r *http.Request, ctx *Ctx, idr *ParsedId, eud string) (stremio.Meta, error) {
	released := time.Now().UTC()

	log := ctx.Log

	meta := stremio.Meta{
		Id:          getWebDLsMetaId(idr.getStoreCode()),
		Type:        ContentTypeOther,
		Name:        "Web Downloads",
		Description: "Web Downloads from Premiumize",
		Released:    &released,
		Videos:      []stremio.MetaVideo{},
	}
	cacheKey := getPMItemsCacheKey(idr.getStoreCode(), ctx.StoreAuthToken)
	if !pmItemsCache.Get(cacheKey, &meta.Videos) {
		params := &stremio_store_webdl.ListWebDLsParams{}
		params.APIKey = ctx.StoreAuthToken
		res, err := stremio_store_webdl.ListWebDLs(params, idr.storeName)
		if err != nil {
			log.Error("failed to list webdls", "error", err, "store.name", idr.storeName)
			return meta, err
		}

		idPrefix := getWebDLsMetaIdPrefix(idr.getStoreCode())

		streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/_/strem/")
		for i := range res.Items {
			item := &res.Items[i]
			if len(item.Files) == 0 {
				continue
			}
			file := item.Files[0]
			if strings.HasPrefix(file.Path, "stremthru/") || !core.HasVideoExtension(file.Name) {
				continue
			}
			streamId := idPrefix + item.Id
			stream := stremio.Stream{
				URL: streamBaseUrl.JoinPath(url.PathEscape(streamId)).String(),
				BehaviorHints: &stremio.StreamBehaviorHints{
					VideoSize: file.Size,
					Filename:  file.Name,
				},
			}
			videoTitle := getMetaPreviewDescriptionForWebDL("", file.Name, true) + "\n📄 " + file.Name
			if config.StoreContentProxy.IsEnabled(string(idr.storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(idr.storeName)) && ctx.IsProxyAuthorized {
				videoTitle = "✨ " + videoTitle
			}
			video := stremio.MetaVideo{
				Id:       streamId,
				Title:    videoTitle,
				Released: item.AddedAt,
				Streams:  []stremio.Stream{stream},
				Episode:  -1,
				Season:   -1,
			}
			meta.Videos = append(meta.Videos, video)
		}
		pmItemsCache.Add(cacheKey, meta.Videos)
	}
	return meta, nil
}
