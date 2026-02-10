package stremio_store

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
	"github.com/MunifTanjim/stremthru/stremio"
)

var rdClient = realdebrid.NewAPIClient(&realdebrid.APIClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("realdebrid")),
	UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
})

var rdDownloadsCache = cache.NewCache[[]stremio.MetaVideo](&cache.CacheConfig{
	Lifetime: 10 * time.Minute,
	Name:     "stremio:store:rd:downloads",
})

func getRDDownloadsCacheKey(idStoreCode, storeToken string) string {
	return getCatalogCacheKey(idStoreCode, storeToken)
}

func getRDWebDLsMeta(r *http.Request, ctx *Ctx, idr *ParsedId) stremio.Meta {
	released := time.Now().UTC()

	log := ctx.Log

	meta := stremio.Meta{
		Id:          getWebDLsMetaId(idr.getStoreCode()),
		Type:        ContentTypeOther,
		Name:        "Web Downloads",
		Description: "Web Downloads from RealDebrid",
		Released:    &released,
		Videos:      []stremio.MetaVideo{},
	}
	cacheKey := getRDDownloadsCacheKey(idr.getStoreCode(), ctx.StoreAuthToken)
	if !rdDownloadsCache.Get(cacheKey, &meta.Videos) {
		storeName := ctx.Store.GetName()

		offset := 0
		hasMore := true
		for hasMore && offset < max_fetch_list_items {
			params := &realdebrid.ListDownloadsParams{
				Limit:  fetch_list_limit,
				Offset: offset,
			}
			params.APIKey = ctx.StoreAuthToken
			res, err := rdClient.ListDownloads(params)
			if err != nil {
				log.Error("failed to list downloads", "error", err, "store.name", storeName)
				break
			}

			shouldCreateProxyLink := config.StoreContentProxy.IsEnabled(string(storeName)) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(storeName)) && ctx.IsProxyAuthorized
			tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
			idPrefix := getWebDLsMetaIdPrefix(idr.getStoreCode())

			for i := range res.Data {
				dl := &res.Data[i]
				if dl.Host == "real-debrid.com" || !core.HasVideoExtension(dl.Filename) {
					continue
				}
				stream := stremio.Stream{
					URL: dl.Download,
					BehaviorHints: &stremio.StreamBehaviorHints{
						VideoSize: dl.Filesize,
						Filename:  dl.Filename,
					},
				}
				videoTitle := getMetaPreviewDescriptionForWebDL(dl.Host, dl.Filename, true) + "\n📄 " + dl.Filename
				if shouldCreateProxyLink {
					if proxyLink, err := shared.CreateProxyLink(r, stream.URL, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, dl.Filename, nil); err == nil {
						stream.URL = proxyLink
						videoTitle = "✨ " + videoTitle
					} else {
						log.Error("failed to create proxy link, skipping file", "error", err, "store.name", storeName, "filename", dl.Filename)
						continue
					}
				}
				video := stremio.MetaVideo{
					Id:       idPrefix + dl.Id,
					Title:    videoTitle,
					Released: dl.Generated,
					Streams:  []stremio.Stream{stream},
					Episode:  -1,
					Season:   -1,
				}
				meta.Videos = append(meta.Videos, video)
			}

			offset += fetch_list_limit
			hasMore = len(res.Data) == fetch_list_limit
			time.Sleep(1 * time.Second)
		}
		rdDownloadsCache.Add(cacheKey, meta.Videos)
	}
	return meta
}
