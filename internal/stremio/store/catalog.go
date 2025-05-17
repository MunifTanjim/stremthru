package stremio_store

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

type CachedCatalogItem struct {
	stremio.MetaPreview
	hash string
}

var catalogCache = func() cache.Cache[[]CachedCatalogItem] {
	c := cache.NewCache[[]CachedCatalogItem](&cache.CacheConfig{
		Lifetime: 10 * time.Minute,
		Name:     "stremio:store:catalog",
	})
	return c
}()

const max_fetch_list_items = 2000
const fetch_list_limit = 500

func getUsenetCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string) []CachedCatalogItem {
	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		offset := 0
		hasMore := true
		for hasMore && offset < max_fetch_list_items {
			params := &stremio_store_usenet.ListNewsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := stremio_store_usenet.ListNews(params, s.GetName())
			if err != nil {
				log.Error("failed to list news", "error", err, "offset", offset)
				break
			}

			for _, item := range res.Items {
				if item.Status == store.MagnetStatusDownloaded {
					cItem := CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.GetLargestFileName(),
						PosterShape: stremio.MetaPosterShapePoster,
					}, item.Hash}
					cItem.Description = getMetaPreviewDescriptionForUsenet(cItem.hash, item.Name, cItem.Name)
					items = append(items, cItem)
				}
			}
			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems
			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
	}

	return items
}

func getWebDLCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string) []CachedCatalogItem {
	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		offset := 0
		hasMore := true
		for hasMore && offset < max_fetch_list_items {
			params := &stremio_store_webdl.ListWebDLsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := stremio_store_webdl.ListWebDLs(params, s.GetName())
			if err != nil {
				log.Error("failed to list webdls", "error", err, "offset", offset)
				break
			}

			for _, item := range res.Items {
				if item.Status == store.MagnetStatusDownloaded {
					cItem := CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						PosterShape: stremio.MetaPosterShapePoster,
					}, item.Hash}
					cItem.Description = getMetaPreviewDescriptionForWebDL(cItem.hash, item.Name, false)
					items = append(items, cItem)
				}
			}
			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems
			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
	}

	return items
}

func getCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string, idr *ParsedId) []CachedCatalogItem {
	if idr.isUsenet {
		return getUsenetCatalogItems(s, storeToken, clientIp, idPrefix)
	}

	if idr.isWebDL {
		return getWebDLCatalogItems(s, storeToken, clientIp, idPrefix)
	}

	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		tInfoItems := []torrent_info.TorrentInfoInsertData{}
		tInfoSource := torrent_info.TorrentInfoSource(s.GetName().Code())

		offset := 0
		hasMore := true
		for hasMore {
			params := &store.ListMagnetsParams{
				Limit:    fetch_list_limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := s.ListMagnets(params)
			if err != nil {
				break
			}

			for _, item := range res.Items {
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
				})
			}
			offset += fetch_list_limit
			hasMore = len(res.Items) == fetch_list_limit && offset < res.TotalItems

			if hasMore && offset >= max_fetch_list_items {
				worker.StoreCrawlerQueue.Queue(worker.StoreCrawlerQueueItem{
					StoreCode:  string(s.GetName().Code()),
					StoreToken: storeToken,
				})
				break
			}

			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
		go torrent_info.Upsert(tInfoItems, "", s.GetName().Code() != store.StoreCodeRealDebrid)
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
	if extraParams := getPathParam(r, "extra"); extraParams != "" {
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

	items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, getIdPrefix(idStoreCode), idr)

	if extra.Search != "" {
		query := strings.ToLower(extra.Search)
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
			if regex.MatchString(strings.ToLower(item.Name)) {
				filteredItems = append(filteredItems, *item)
			}
		}
		items = filteredItems
	}

	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	hashes := make([]string, len(items))
	for i := range items {
		item := &items[i]
		hashes[i] = item.hash
	}

	includeRDDownlodsMetaPreview := ud.EnableWebDL && idr.storeCode == store.StoreCodeRealDebrid

	count := len(hashes)
	if includeRDDownlodsMetaPreview {
		count += 1
	}

	res.Metas = make([]stremio.MetaPreview, 0, count)

	if includeRDDownlodsMetaPreview {
		res.Metas = append(res.Metas, stremio.MetaPreview{
			Id:     getRDWebDLsId(idStoreCode),
			Type:   ContentTypeOther,
			Name:   "Web Downloads",
			Poster: "https://emojiapi.dev/api/v1/inbox_tray/256.png",
		})
	}

	stremIdByHash, err := torrent_stream.GetStremIdByHashes(hashes)
	if err != nil {
		log.Error("failed to get strem id by hashes", "error", err)
	}
	for i := range items {
		item := &items[i]
		if stremId := stremIdByHash.Get(item.hash); stremId != "" {
			stremId, _, _ = strings.Cut(stremId, ":")
			item.Poster = getPosterUrl(stremId)
		}
		res.Metas = append(res.Metas, item.MetaPreview)
	}

	SendResponse(w, r, 200, res)
}
