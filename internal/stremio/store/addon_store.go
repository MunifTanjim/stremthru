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
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/sahilm/fuzzy"
)

type UserData struct {
	StoreName  string `json:"store_name"`
	StoreToken string `json:"store_token"`
	encoded    string `json:"-"`
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

func (uderr userDataError) Error() string {
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

func (ud UserData) GetRequestContext(r *http.Request) (*context.RequestContext, *userDataError) {
	ctx := &context.RequestContext{}

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

	if !ctx.IsProxyAuthorized {
		ctx.ClientIP = core.GetClientIP(r)
	}

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}

	if IsMethod(r, http.MethodGet) {
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

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	manifest := getManifest(ud)

	SendResponse(w, 200, manifest)
}

func getTemplateData() *configure.TemplateData {
	return &configure.TemplateData{
		Title: "StremThru Store",
		Configs: []configure.Config{
			configure.Config{
				Key:     "store_name",
				Type:    "select",
				Default: "",
				Title:   "Store Name",
				Options: []configure.ConfigOption{
					configure.ConfigOption{Value: "", Label: "StremThru"},
					configure.ConfigOption{Value: "realdebrid", Label: "RealDebrid"},
					configure.ConfigOption{Value: "alldebrid", Label: "AllDebrid"},
					configure.ConfigOption{Value: "debridlink", Label: "DebridLink"},
					configure.ConfigOption{Value: "premiumize", Label: "Premiumize"},
					configure.ConfigOption{Value: "torbox", Label: "TorBox"},
				},
				Required: false,
			},
			configure.Config{
				Key:      "store_token",
				Type:     "password",
				Default:  "",
				Title:    "Store Token",
				Required: true,
			},
		},
	}
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
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
			SendError(w, err)
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

	ctx, uderr := ud.GetRequestContext(r)
	if uderr != nil {
		if uderr.storeName != "" {
			name_config.Error = uderr.storeName
		}
		if uderr.storeToken != "" {
			token_config.Error = uderr.storeToken
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
			token_config.Error = "Invalid Token"
		} else if user.SubscriptionStatus != store.UserSubscriptionStatusPremium {
			token_config.Error = "Not Premium"
		}
	}

	if td.HasError() {
		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, err)
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
	Type   string
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
			if type_ := q.Get("type"); type_ != "" {
				extra.Type = type_
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

func getCatalogCacheKey(ctx *context.RequestContext) string {
	return string(ctx.Store.GetName().Code()) + ":" + ctx.StoreAuthToken
}

func getStoreActionMetaPreview() stremio.MetaPreview {
	meta := stremio.MetaPreview{
		Id:   STORE_ACTION_ID,
		Type: ContentTypeOther,
		Name: "StremThru Store Actions",
	}
	return meta
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w)
		return
	}

	if catalogId := getId(r); catalogId != CATALOG_ID {
		shared.ErrorBadRequest(r, "unsupported catalog id: "+catalogId).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		shared.ErrorBadRequest(r, "").Send(w)
		return
	}

	extra := getExtra(r)

	res := stremio.CatalogHandlerResponse{
		Metas: []stremio.MetaPreview{},
	}

	if extra.Type == CatalogTypeAction {
		res.Metas = append(res.Metas, getStoreActionMetaPreview())
		SendResponse(w, 200, res)
		return
	}

	items := []stremio.MetaPreview{}

	cacheKey := getCatalogCacheKey(ctx)
	if !catalogCache.Get(cacheKey, &items) {
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
						Id:          ID_PREFIX + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						Description: item.Hash,
					})
				}
			}
			offset += limit
			hasMore = len(res.Items) == limit && offset < res.TotalItems
			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
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

	res.Metas = items

	SendResponse(w, 200, res)
}

func getStoreActionMeta(r *http.Request, encodedUserData string) stremio.Meta {
	released := time.Now().UTC()
	meta := stremio.Meta{
		Id:          STORE_ACTION_ID,
		Type:        ContentTypeOther,
		Name:        "StremThru Store Actions",
		Description: "Actions for StremThru Store",
		Released:    released,
		Videos: []stremio.MetaVideo{
			stremio.MetaVideo{
				Id:       STORE_ACTION_ID_PREFIX + "clear_cache",
				Title:    "Clear Cache",
				Released: released,
				Streams: []stremio.Stream{stremio.Stream{
					URL:         ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + encodedUserData + "/_/action/" + STORE_ACTION_ID_PREFIX + "clear_cache").String(),
					Name:        "Clear Cache",
					Description: "Clear Cached Data for StremThru Store",
				}},
			},
		},
	}
	return meta
}

func handleMeta(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w)
		return
	}

	id := getId(r)
	if !strings.HasPrefix(id, ID_PREFIX) {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		shared.ErrorBadRequest(r, "").Send(w)
		return
	}

	if id == STORE_ACTION_ID {
		eud, err := ud.GetEncoded()
		if err != nil {
			SendError(w, err)
			return
		}

		res := stremio.MetaHandlerResponse{
			Meta: getStoreActionMeta(r, eud),
		}

		SendResponse(w, 200, res)
		return
	}

	params := &store.GetMagnetParams{
		Id: strings.TrimPrefix(id, ID_PREFIX),
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, err)
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

	for _, f := range magnet.Files {
		videoId := id + ":" + strconv.Itoa(f.Idx)
		res.Meta.Videos = append(res.Meta.Videos, stremio.MetaVideo{
			Id:        videoId,
			Title:     f.Name,
			Available: true,
			Released:  magnet.AddedAt,
		})
	}

	SendResponse(w, 200, res)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w)
		return
	}

	videoIdWithIdx := getId(r)
	if !strings.HasPrefix(videoIdWithIdx, ID_PREFIX) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithIdx).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		shared.ErrorBadRequest(r, "").Send(w)
		return
	}

	res := stremio.StreamHandlerResponse{
		Streams: []stremio.Stream{},
	}

	videoId := strings.TrimPrefix(videoIdWithIdx, ID_PREFIX)
	videoId, fileIdxStr, _ := strings.Cut(videoId, ":")
	if fileIdxStr == "" {
		fileIdxStr = "0"
	}
	fileIdx, err := strconv.Atoi(fileIdxStr)
	if err != nil {
		SendError(w, err)
		return
	}

	params := &store.GetMagnetParams{
		Id: videoId,
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, err)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, err)
		return
	}

	url := ExtractRequestBaseURL(r)
	for _, f := range magnet.Files {
		if f.Idx == fileIdx {
			res.Streams = append(res.Streams, stremio.Stream{
				URL:         url.JoinPath("/stremio/store/" + eud + "/_/strem/" + videoIdWithIdx).String(),
				Name:        magnet.Name,
				Description: f.Name,
			})
		}
	}

	SendResponse(w, 200, res)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	actionId := r.PathValue("actionId")
	if !strings.HasPrefix(actionId, STORE_ACTION_ID_PREFIX) {
		shared.ErrorBadRequest(r, "unsupported id: "+actionId).Send(w)
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		w.WriteHeader(204)
		w.Write([]byte{})
		return
	}

	switch strings.TrimPrefix(actionId, STORE_ACTION_ID_PREFIX) {
	case "clear_cache":
		cacheKey := getCatalogCacheKey(ctx)
		catalogCache.Remove(cacheKey)
	}

	w.WriteHeader(204)
	w.Write([]byte{})
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) || !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	videoIdWithIdx := r.PathValue("videoId")
	if !strings.HasPrefix(videoIdWithIdx, ID_PREFIX) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithIdx).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil || ctx.Store == nil {
		shared.ErrorBadRequest(r, "").Send(w)
		return
	}

	videoId := strings.TrimPrefix(videoIdWithIdx, ID_PREFIX)
	videoId, fileIdxStr, _ := strings.Cut(videoId, ":")
	if fileIdxStr == "" {
		fileIdxStr = "0"
	}
	fileIdx, err := strconv.Atoi(fileIdxStr)
	if err != nil {
		shared.ErrorNotFound(r).Send(w)
		return
	}

	params := &store.GetMagnetParams{
		Id: videoId,
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, err)
		return
	}

	url := ""
	for _, f := range magnet.Files {
		if f.Idx == fileIdx {
			url = f.Link
			break
		}
	}

	if url == "" {
		shared.ErrorNotFound(r).Send(w)
		return
	}

	stLink, err := shared.GenerateStremThruLink(r, ctx, url)
	if err != nil {
		SendError(w, err)
		return
	}

	http.Redirect(w, r, stLink.Link, http.StatusFound)
}

func AddStremioStoreEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/stremio/store/manifest.json", handleManifest)
	mux.HandleFunc("/stremio/store/{userData}/manifest.json", handleManifest)

	mux.HandleFunc("/stremio/store/configure", handleConfigure)
	mux.HandleFunc("/stremio/store/{userData}/configure", handleConfigure)

	mux.HandleFunc("/stremio/store/{userData}/catalog/{contentType}/{idJson}", handleCatalog)
	mux.HandleFunc("/stremio/store/{userData}/catalog/{contentType}/{id}/{extraJson}", handleCatalog)

	mux.HandleFunc("/stremio/store/{userData}/meta/{contentType}/{idJson}", handleMeta)

	mux.HandleFunc("/stremio/store/{userData}/stream/{contentType}/{idJson}", handleStream)

	mux.HandleFunc("/stremio/store/{userData}/_/action/{actionId}", handleAction)
	mux.HandleFunc("/stremio/store/{userData}/_/strem/{videoId}", handleStrem)
}
