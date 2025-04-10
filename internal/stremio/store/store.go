package stremio_store

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/store/video"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/sahilm/fuzzy"
)

type UserData struct {
	StoreName  string `json:"store_name"`
	StoreToken string `json:"store_token"`
	encoded    string `json:"-"`
	storeCode  string `json:"-"`
}

func (ud *UserData) getStoreCode() string {
	if ud.storeCode == "" {
		switch ud.StoreName {
		case "":
			ud.storeCode = "st"
		case "stremthru":
			ud.storeCode = "st"
		default:
			ud.storeCode = string(store.StoreName(ud.StoreName).Code())
		}
	}
	return ud.storeCode
}

func (ud UserData) HasRequiredValues() bool {
	return ud.StoreToken != ""
}

func (ud UserData) GetEncoded() (string, error) {
	if ud.encoded != "" {
		return ud.encoded, nil
	}

	blob, err := json.Marshal(ud)
	if err != nil {
		return "", err
	}
	return core.Base64Encode(string(blob)), nil
}

type userDataError struct {
	storeToken string
	storeName  string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	if uderr.storeName != "" {
		str.WriteString("store_name: ")
		str.WriteString(uderr.storeName)
		hasSome = true
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.storeToken != "" {
		str.WriteString("store_token: ")
		str.WriteString(uderr.storeToken)
	}
	return str.String()
}

func (ud UserData) GetRequestContext(r *http.Request) (*context.StoreContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &context.StoreContext{
		Log: rCtx.Log,
	}

	storeName := ud.StoreName
	storeToken := ud.StoreToken
	if storeName == "" {
		user, err := core.ParseBasicAuth(storeToken)
		if err != nil {
			return ctx, &userDataError{storeToken: err.Error()}
		}
		password := config.ProxyAuthPassword.GetPassword(user.Username)
		if password != "" && password == user.Password {
			ctx.IsProxyAuthorized = true
			ctx.ProxyAuthUser = user.Username
			ctx.ProxyAuthPassword = user.Password

			storeName = config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser)
			storeToken = config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, storeName)
		}
	}

	if storeToken != "" {
		ctx.Store = shared.GetStore(storeName)
		ctx.StoreAuthToken = storeToken
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		data.encoded = r.PathValue("userData")
		if data.encoded == "" {
			return data, nil
		}
		blob, err := core.Base64DecodeToByte(data.encoded)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(blob, data)
		return data, err
	}

	if IsMethod(r, http.MethodPost) {
		data.StoreName = r.FormValue("store_name")
		data.StoreToken = r.FormValue("store_token")
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	return data, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/store/configure", http.StatusFound)
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	manifest := GetManifest(r, ud)

	SendResponse(w, r, 200, manifest)
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData()
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "store_name":
			conf.Default = ud.StoreName
		case "store_token":
			conf.Default = ud.StoreToken
		}
	}

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/manifest.json").String()
			}
		}

		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	var name_config *configure.Config
	var token_config *configure.Config
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "store_name":
			name_config = conf
		case "store_token":
			token_config = conf
		}
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			if uderr.storeName != "" {
				name_config.Error = uderr.storeName
			}
			if uderr.storeToken != "" {
				token_config.Error = uderr.storeToken
			}
		} else {
			SendError(w, r, err)
			return
		}
	}

	if ctx.Store == nil {
		if ud.StoreName == "" {
			token_config.Error = "Invalid Token"
		} else {
			name_config.Error = "Invalid Store"
		}
	} else if token_config.Error == "" {
		params := &store.GetUserParams{}
		params.APIKey = ctx.StoreAuthToken
		user, err := ctx.Store.GetUser(params)
		if err != nil {
			LogError(r, "failed to get user", err)
			token_config.Error = "Invalid Token"
		} else if user.SubscriptionStatus == store.UserSubscriptionStatusExpired {
			token_config.Error = "Subscription Expired"
		}
	}

	if td.HasError() {
		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	url := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/configure")
	q := url.Query()
	q.Set("try_install", "1")
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}

func getContentType(r *http.Request) (string, *core.APIError) {
	contentType := r.PathValue("contentType")
	if contentType != ContentTypeOther {
		return "", shared.ErrorBadRequest(r, "unsupported type: "+contentType)
	}
	return contentType, nil
}

func getPathParam(r *http.Request, name string) string {
	if value := r.PathValue(name + "Json"); value != "" {
		return strings.TrimSuffix(value, ".json")
	}
	return r.PathValue(name)
}

func getId(r *http.Request) string {
	return getPathParam(r, "id")
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

type CatalogSearchDataset struct {
	items []stremio.MetaPreview
}

func (d CatalogSearchDataset) String(i int) string {
	return d.items[i].Name
}

func (d CatalogSearchDataset) Len() int {
	return len(d.items)
}

var catalogCache = func() cache.Cache[[]stremio.MetaPreview] {
	c := cache.NewCache[[]stremio.MetaPreview](&cache.CacheConfig{
		Lifetime: 5 * time.Minute,
		Name:     "stremio:store:catalog",
	})
	return c
}()

func getCatalogCacheKey(ctx *context.StoreContext) string {
	return string(ctx.Store.GetName().Code()) + ":" + ctx.StoreAuthToken
}

func getStoreActionMetaPreview(storeCode string) stremio.MetaPreview {
	meta := stremio.MetaPreview{
		Id:   getStoreActionId(storeCode),
		Type: ContentTypeOther,
		Name: "StremThru Store Actions",
	}
	return meta
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if catalogId := getId(r); catalogId != getCatalogId(ud.getStoreCode()) {
		shared.ErrorBadRequest(r, "unsupported catalog id: "+catalogId).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r)
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

	if extra.Genre == CatalogGenreStremThru {
		res.Metas = append(res.Metas, getStoreActionMetaPreview(ud.getStoreCode()))
		SendResponse(w, r, 200, res)
		return
	}

	items := []stremio.MetaPreview{}

	cacheKey := getCatalogCacheKey(ctx)
	if !catalogCache.Get(cacheKey, &items) {
		idPrefix := getIdPrefix(ud.getStoreCode())

		tInfoItems := []torrent_info.TorrentInfoInsertData{}
		tInfoSource := torrent_info.TorrentInfoSource(ctx.Store.GetName().Code())

		limit := 500
		offset := 0
		hasMore := true
		for hasMore && offset < 2000 {
			params := &store.ListMagnetsParams{
				Limit:  limit,
				Offset: offset,
			}
			params.APIKey = ctx.StoreAuthToken
			res, err := ctx.Store.ListMagnets(params)
			if err != nil {
				break
			}

			for _, item := range res.Items {
				if item.Status == store.MagnetStatusDownloaded {
					items = append(items, stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						Description: item.Hash,
					})
				}
				tInfoItems = append(tInfoItems, torrent_info.TorrentInfoInsertData{
					Hash:         item.Hash,
					TorrentTitle: item.Name,
					Size:         item.Size,
					Source:       tInfoSource,
				})
			}
			offset += limit
			hasMore = len(res.Items) == limit && offset < res.TotalItems
			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
		go torrent_info.Upsert(tInfoItems, "", ctx.Store.GetName().Code() != store.StoreCodeRealDebrid)
	}

	if extra.Search != "" {
		matches := fuzzy.FindFrom(extra.Search, &CatalogSearchDataset{items: items})
		filteredItems := make([]stremio.MetaPreview, len(matches))
		for i := range matches {
			filteredItems[i] = items[matches[i].Index]
		}
		items = filteredItems
	}

	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	if len(items) > 0 {
		res.Metas = items
	}

	SendResponse(w, r, 200, res)
}

func getStoreActionMeta(r *http.Request, storeCode string, encodedUserData string) stremio.Meta {
	released := time.Now().UTC()
	meta := stremio.Meta{
		Id:          getStoreActionId(storeCode),
		Type:        ContentTypeOther,
		Name:        "StremThru Store Actions",
		Description: "Actions for StremThru Store",
		Released:    released,
		Videos: []stremio.MetaVideo{
			{
				Id:       getStoreActionIdPrefix(storeCode) + "clear_cache",
				Title:    "Clear Cache",
				Released: released,
				Streams: []stremio.Stream{
					{
						URL:         ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + encodedUserData + "/_/action/" + getStoreActionIdPrefix(storeCode) + "clear_cache").String(),
						Name:        "Clear Cache",
						Description: "Clear Cached Data for StremThru Store",
					},
				},
			},
		},
	}
	return meta
}

func handleMeta(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idPrefix := getIdPrefix(ud.getStoreCode())

	id := getId(r)
	if !strings.HasPrefix(id, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "").Send(w, r)
		return
	}

	if id == getStoreActionId(ud.getStoreCode()) {
		eud, err := ud.GetEncoded()
		if err != nil {
			SendError(w, r, err)
			return
		}

		res := stremio.MetaHandlerResponse{
			Meta: getStoreActionMeta(r, ud.getStoreCode(), eud),
		}

		SendResponse(w, r, 200, res)
		return
	}

	params := &store.GetMagnetParams{
		Id: strings.TrimPrefix(id, idPrefix),
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	res := stremio.MetaHandlerResponse{
		Meta: stremio.Meta{
			Id:          id,
			Type:        ContentTypeOther,
			Name:        magnet.Name,
			Description: magnet.Hash,
			Released:    magnet.AddedAt,
			Videos:      []stremio.MetaVideo{},
		},
	}

	tInfo := torrent_info.TorrentInfoInsertData{
		Hash:         magnet.Hash,
		TorrentTitle: magnet.Name,
		Size:         magnet.Size,
		Source:       torrent_info.TorrentInfoSource(ctx.Store.GetName().Code()),
		Files:        []torrent_info.TorrentInfoInsertDataFile{},
	}

	for _, f := range magnet.Files {
		videoId := id + ":" + url.PathEscape(f.Link)
		res.Meta.Videos = append(res.Meta.Videos, stremio.MetaVideo{
			Id:        videoId,
			Title:     f.Name,
			Available: true,
			Released:  magnet.AddedAt,
		})
		tInfo.Files = append(tInfo.Files, torrent_info.TorrentInfoInsertDataFile{
			Name: f.Name,
			Idx:  f.Idx,
			Size: f.Size,
		})
	}

	go torrent_info.Upsert([]torrent_info.TorrentInfoInsertData{tInfo}, "", ctx.Store.GetName().Code() != store.StoreCodeRealDebrid)

	SendResponse(w, r, 200, res)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idPrefix := getIdPrefix(ud.getStoreCode())

	videoIdWithLink := getId(r)
	if !strings.HasPrefix(videoIdWithLink, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "").Send(w, r)
		return
	}

	res := stremio.StreamHandlerResponse{
		Streams: []stremio.Stream{},
	}

	videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
	videoId, escapedLink, _ := strings.Cut(videoId, ":")
	link, err := url.PathUnescape(escapedLink)
	if err != nil {
		LogError(r, "failed to parse link", err)
		SendError(w, r, err)
		return
	}

	params := &store.GetMagnetParams{
		Id: videoId,
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	baseUrl := ExtractRequestBaseURL(r)
	for _, f := range magnet.Files {
		if f.Link == link {
			streamId := idPrefix + videoId + ":" + link
			res.Streams = append(res.Streams, stremio.Stream{
				URL:         baseUrl.JoinPath("/stremio/store/" + eud + "/_/strem/" + url.PathEscape(streamId)).String(),
				Name:        magnet.Name,
				Description: f.Name,
			})
		}
	}

	SendResponse(w, r, 200, res)
}

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

	storeActionIdPrefix := getStoreActionIdPrefix(ud.getStoreCode())

	actionId := r.PathValue("actionId")
	if !strings.HasPrefix(actionId, storeActionIdPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+actionId).Send(w, r)
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		store_video.Redirect("500", w, r)
		return
	}

	switch strings.TrimPrefix(actionId, storeActionIdPrefix) {
	case "clear_cache":
		cacheKey := getCatalogCacheKey(ctx)
		catalogCache.Remove(cacheKey)
	}

	store_video.Redirect("200", w, r)
}

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

	idPrefix := getIdPrefix(ud.getStoreCode())

	videoIdWithLink := r.PathValue("videoId")
	if !strings.HasPrefix(videoIdWithLink, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r)
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

	stLink, err := shared.GenerateStremThruLink(r, ctx, url)
	if err != nil {
		LogError(r, "failed to generate stremthru link", err)
		store_video.Redirect("500", w, r)
		return
	}

	http.Redirect(w, r, stLink.Link, http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioStoreEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/catalog/{contentType}/{idJson}", withCors(handleCatalog))
	router.HandleFunc("/{userData}/catalog/{contentType}/{id}/{extraJson}", withCors(handleCatalog))

	router.HandleFunc("/{userData}/meta/{contentType}/{idJson}", withCors(handleMeta))

	router.HandleFunc("/{userData}/stream/{contentType}/{idJson}", withCors(handleStream))

	router.HandleFunc("/{userData}/_/action/{actionId}", withCors(handleAction))
	router.HandleFunc("/{userData}/_/strem/{videoId}", withCors(handleStrem))

	mux.Handle("/stremio/store/", http.StripPrefix("/stremio/store", commonMiddleware(router)))
}
