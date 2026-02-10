package stremio_store

import (
	"net/http"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/stremio"
)

var adLinksCache = cache.NewCache[[]stremio.MetaVideo](&cache.CacheConfig{
	Lifetime: 10 * time.Minute,
	Name:     "stremio:store:ad:links",
})

func getADLinksCacheKey(idStoreCode, storeToken string) string {
	return getCatalogCacheKey(idStoreCode, storeToken)
}

func getADWebDLsMeta(r *http.Request, ctx *Ctx, idr *ParsedId, eud string) stremio.Meta {
	log := ctx.Log

	released := time.Now().UTC()

	meta := stremio.Meta{
		Id:          getWebDLsMetaId(idr.getStoreCode()),
		Type:        ContentTypeOther,
		Name:        "Web Downloads",
		Description: "Web Downloads from AllDebrid",
		Released:    &released,
		Videos:      []stremio.MetaVideo{},
	}
	cacheKey := getADLinksCacheKey(idr.getStoreCode(), ctx.StoreAuthToken)
	if !adLinksCache.Get(cacheKey, &meta.Videos) {
		params := &stremio_store_webdl.ListWebDLsParams{}
		params.APIKey = ctx.StoreAuthToken
		res, err := stremio_store_webdl.ListWebDLs(params, idr.storeName)
		if err != nil {
			log.Error("failed to list webdls", "error", err, "store.name", idr.storeName)
			return meta
		}

		storeName := ctx.Store.GetName()
		shouldCreateProxyLink := config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) && ctx.IsProxyAuthorized
		tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
		idPrefix := getWebDLsMetaIdPrefix(idr.getStoreCode())

		streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/_/strem/")
		for i := range res.Items {
			dl := &res.Items[i]

			stream := stremio.Stream{
				BehaviorHints: &stremio.StreamBehaviorHints{
					VideoSize: dl.Size,
					Filename:  dl.Name,
				},
			}

			videoId := idPrefix + dl.Id
			isDirectLink := false
			if len(dl.Files) == 0 {
				stream.URL = streamBaseUrl.JoinPath(url.PathEscape(videoId)).String()
			} else {
				file := dl.Files[0]
				if !core.HasVideoExtension(file.Name) {
					continue
				}
				isDirectLink = true
				stream.URL = file.Link
				stream.BehaviorHints.VideoSize = file.Size
				stream.BehaviorHints.Filename = file.Name
			}

			videoTitle := getMetaPreviewDescriptionForWebDL("", dl.Name, true) + "\n📄 " + stream.BehaviorHints.Filename

			if shouldCreateProxyLink {
				videoTitle = "✨ " + videoTitle
				if isDirectLink {
					if proxyLink, err := shared.CreateProxyLink(r, stream.URL, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, stream.BehaviorHints.Filename); err == nil {
						stream.URL = proxyLink
					} else {
						log.Error("failed to create proxy link, skipping file", "error", err, "store.name", storeName, "filename", stream.BehaviorHints.Filename)
						continue
					}
				}
			}

			video := stremio.MetaVideo{
				Id:       videoId,
				Title:    videoTitle,
				Released: dl.AddedAt,
				Streams:  []stremio.Stream{stream},
				Episode:  -1,
				Season:   -1,
			}
			meta.Videos = append(meta.Videos, video)
		}

		rdDownloadsCache.Add(cacheKey, meta.Videos)
	}
	return meta
}
