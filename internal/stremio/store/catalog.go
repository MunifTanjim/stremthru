package stremio_store

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

type CachedCatalogItem struct {
	stremio.MetaPreview
	Hash string
}

var catalogCache = cache.NewCache[[]CachedCatalogItem](&cache.CacheConfig{
	Lifetime:      config.Stremio.Store.CatalogCacheTime,
	Name:          "stremio:store:catalog",
	LocalCapacity: 2048,
})

var max_fetch_list_items = config.Stremio.Store.CatalogItemLimit

const fetch_list_limit = 500

func getUsenetCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string, log *logger.Logger) []CachedCatalogItem {
	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		storeName := s.GetName()

		offset := 0
		hasMore := true
		for hasMore && offset < max_fetch_list_items {
			start := time.Now()
			params := &stremio_store_usenet.ListNewsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := stremio_store_usenet.ListNews(params, storeName)
			if err != nil {
				log.Error("failed to list news", "error", err, "duration", time.Since(start).String(), "store.name", storeName, "offset", offset)
				break
			}
			count := len(res.Items)
			log.Debug("fetched news", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			start = time.Now()
			for i := range res.Items {
				item := &res.Items[i]
				if item.Status == store.MagnetStatusDownloaded {
					cItem := CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						PosterShape: stremio.MetaPosterShapePoster,
					}, item.Hash}
					cItem.Description = getMetaPreviewDescriptionForUsenet(cItem.Hash, item.Name, item.GetLargestFileName())
					items = append(items, cItem)
				}
			}
			log.Debug("processed news", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems

			time.Sleep(500 * time.Millisecond)
		}
		catalogCache.Add(cacheKey, items)
	}

	return items
}

func getWebDLCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string, log *logger.Logger) []CachedCatalogItem {
	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		storeName := s.GetName()

		offset := 0
		hasMore := true
		for hasMore && offset < max_fetch_list_items {
			start := time.Now()
			params := &stremio_store_webdl.ListWebDLsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := stremio_store_webdl.ListWebDLs(params, storeName)
			if err != nil {
				log.Error("failed to list webdls", "error", err, "duration", time.Since(start).String(), "store.name", storeName, "offset", offset)
				break
			}
			count := len(res.Items)
			log.Debug("fetched webdls", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			start = time.Now()
			for i := range res.Items {
				item := &res.Items[i]
				if item.Status == store.MagnetStatusDownloaded {
					cItem := CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						PosterShape: stremio.MetaPosterShapePoster,
					}, item.Hash}
					cItem.Description = getMetaPreviewDescriptionForWebDL(cItem.Hash, item.Name, false)
					items = append(items, cItem)
				}
			}
			log.Debug("processed webdls", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems

			time.Sleep(500 * time.Millisecond)
		}
		catalogCache.Add(cacheKey, items)
	}

	return items
}

func getCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string, idr *ParsedId, log *logger.Logger) []CachedCatalogItem {
	if idr.isUsenet {
		return getUsenetCatalogItems(s, storeToken, clientIp, idPrefix, log)
	}

	if idr.isWebDL {
		return getWebDLCatalogItems(s, storeToken, clientIp, idPrefix, log)
	}

	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		storeName := s.GetName()
		storeCode := storeName.Code()

		tInfoItems := []torrent_info.TorrentInfoInsertData{}
		tInfoSource := torrent_info.TorrentInfoSource(storeCode)

		offset := 0
		hasMore := true
		for hasMore {
			start := time.Now()
			params := &store.ListMagnetsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := s.ListMagnets(params)
			if err != nil {
				log.Error("failed to list magnets", "error", err, "duration", time.Since(start).String(), "store.name", storeName, "offset", offset)
				break
			}
			count := len(res.Items)
			log.Debug("fetched magnets", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			start = time.Now()
			for i := range res.Items {
				item := &res.Items[i]
				if item.Status == store.MagnetStatusDownloaded {
					items = append(items, CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						Description: getMetaPreviewDescriptionForTorrent(item.Hash, item.Name),
						PosterShape: stremio.MetaPosterShapePoster,
					}, item.Hash})
				}
				tInfoItems = append(tInfoItems, torrent_info.TorrentInfoInsertData{
					Hash:         item.Hash,
					TorrentTitle: item.Name,
					Size:         item.Size,
					Source:       tInfoSource,
					Private:      item.Private,
				})
			}
			log.Debug("processed magnets", "duration", time.Since(start).String(), "store.name", storeName, "offset", offset, "count", count)

			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems

			if hasMore && offset >= max_fetch_list_items {
				worker_queue.StoreCrawlerQueue.Queue(worker_queue.StoreCrawlerQueueItem{
					StoreCode:  string(storeCode),
					StoreToken: storeToken,
				})
				break
			}

			time.Sleep(500 * time.Millisecond)
		}
		catalogCache.Add(cacheKey, items)
		go torrent_info.Upsert(tInfoItems, "", storeCode != store.StoreCodeRealDebrid)
	}

	return items
}

type ExtraData struct {
	Search string
	Skip   int
	Genre  string
}

func getExtra(r *http.Request) *ExtraData {
	extra := &ExtraData{}
	if extraParams := GetPathValue(r, "extra"); extraParams != "" {
		if q, err := url.ParseQuery(extraParams); err == nil {
			if search := q.Get("search"); search != "" {
				extra.Search = search
			}
			if skipStr := q.Get("skip"); skipStr != "" {
				if skip, err := strconv.Atoi(skipStr); err == nil {
					extra.Skip = skip
				}
			}
			if genre := q.Get("genre"); genre != "" {
				extra.Genre = genre
			}
		}
	}
	return extra
}

func getStoreActionMetaPreview(storeCode string) stremio.MetaPreview {
	meta := stremio.MetaPreview{
		Id:   getStoreActionId(storeCode),
		Type: ContentTypeOther,
		Name: "StremThru Store Actions",
	}
	return meta
}

func getCatalogCacheKey(idPrefix, storeToken string) string {
	return idPrefix + storeToken
}

var whitespacesRegex = regexp.MustCompile(`\s+`)

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	catalogId := getId(r)
	idr, err := parseId(catalogId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if catalogId != getCatalogId(idr.getStoreCode()) {
		shared.ErrorBadRequest(r, "unsupported catalog id: "+catalogId).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "").Send(w, r)
		return
	}

	log := ctx.Log

	extra := getExtra(r)

	res := stremio.CatalogHandlerResponse{
		Metas: []stremio.MetaPreview{},
	}

	idStoreCode := idr.getStoreCode()

	if extra.Genre == CatalogGenreStremThru {
		res.Metas = append(res.Metas, getStoreActionMetaPreview(idStoreCode))
		SendResponse(w, r, 200, res)
		return
	}

	items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, getIdPrefix(idStoreCode), idr, log)

	if extra.Search != "" {
		start := time.Now()
		normalizer := util.NewStringNormalizer()
		query := normalizer.Normalize(extra.Search)
		parts := whitespacesRegex.Split(query, -1)
		includeStoreActions := false
		for i, part := range parts {
			if !includeStoreActions && (part == "stremthru" || part == "st") {
				includeStoreActions = true
			}
			parts[i] = regexp.QuoteMeta(part)
		}
		regex, err := regexp.Compile(strings.Join(parts, ".*"))
		if err != nil {
			SendError(w, r, err)
			return
		}
		filteredItems := []CachedCatalogItem{}
		if includeStoreActions {
			filteredItems = append(filteredItems, CachedCatalogItem{
				MetaPreview: getStoreActionMetaPreview(idStoreCode),
			})
		}
		for i := range items {
			item := &items[i]
			if regex.MatchString(normalizer.Normalize(item.Name)) {
				filteredItems = append(filteredItems, *item)
			}
		}
		items = filteredItems
		log.Debug("filtered catalog items", "duration", time.Since(start).String())
	}

	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	hashes := make([]string, len(items))
	for i := range items {
		item := &items[i]
		hashes[i] = item.Hash
	}

	includeWebDLsMetaPreview := ud.EnableWebDL && (idr.storeCode == store.StoreCodeRealDebrid || idr.storeCode == store.StoreCodePremiumize || idr.storeCode == store.StoreCodeAllDebrid)

	count := len(hashes)
	if includeWebDLsMetaPreview {
		count += 1
	}

	res.Metas = make([]stremio.MetaPreview, 0, count)

	if includeWebDLsMetaPreview && extra.Skip == 0 {
		res.Metas = append(res.Metas, stremio.MetaPreview{
			Id:          getWebDLsMetaId(idStoreCode),
			Type:        ContentTypeOther,
			Name:        "Web Downloads",
			Description: "Web Downloads for " + strings.ToUpper(string(idr.storeCode)),
			Poster:      "https://emojiapi.dev/api/v1/inbox_tray/256.png",
		})
	}

	stremIdByHash, err := torrent_stream.GetStremIdByHashes(hashes)
	if err != nil {
		log.Error("failed to get strem id by hashes", "error", err)
	}
	for i := range items {
		item := &items[i]
		if stremId := stremIdByHash.Get(item.Hash); stremId != "" {
			stremId, _, _ = strings.Cut(stremId, ":")
			item.Poster = stremio_shared.GetCinemetaPosterURL(stremId)
			item.Background = stremio_shared.GetCinemetaBackgroundURL(stremId)
		}
		res.Metas = append(res.Metas, item.MetaPreview)
	}

	SendResponse(w, r, 200, res)
}
