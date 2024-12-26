package stremio_wrap

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
	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var addon = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

type UserData struct {
	ManifestURL string   `json:"manifest_url"`
	StoreName   string   `json:"store"`
	StoreToken  string   `json:"token"`
	encoded     string   `json:"-"`
	baseUrl     *url.URL `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	return ud.ManifestURL != "" && ud.StoreToken != ""
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
	manifestUrl string
	store       string
	token       string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	if uderr.manifestUrl != "" {
		str.WriteString("manifest_url: ")
		str.WriteString(uderr.manifestUrl)
		hasSome = true
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.store != "" {
		str.WriteString("store: ")
		str.WriteString(uderr.store)
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.token != "" {
		str.WriteString("token: ")
		str.WriteString(uderr.token)
	}
	return str.String()
}

func (ud UserData) GetRequestContext(r *http.Request) (*context.RequestContext, error) {
	ctx := &context.RequestContext{}

	if ud.baseUrl == nil {
		return ctx, &userDataError{manifestUrl: "Invalid Manifest URL"}
	}

	storeName := ud.StoreName
	storeToken := ud.StoreToken
	if storeName == "" {
		auth, err := core.ParseBasicAuth(storeToken)
		if err != nil {
			return ctx, &userDataError{token: err.Error()}
		}
		password := config.ProxyAuthPassword.GetPassword(auth.Username)
		if password != "" && password == auth.Password {
			ctx.IsProxyAuthorized = true
			ctx.ProxyAuthUser = auth.Username
			ctx.ProxyAuthPassword = auth.Password

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
		if err != nil {
			return nil, err
		}
	}

	if IsMethod(r, http.MethodPost) {
		data.ManifestURL = r.FormValue("manifest_url")
		data.StoreName = r.FormValue("store")
		data.StoreToken = r.FormValue("token")
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	if data.ManifestURL != "" {
		if baseUrl, err := url.Parse(data.ManifestURL); err == nil {
			baseUrl.Path = strings.TrimSuffix(baseUrl.Path, "/manifest.json")
			data.baseUrl = baseUrl
		}
	}

	return data, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/wrap/configure", http.StatusFound)
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

	res, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl})
	if err != nil {
		SendError(w, err)
		return
	}

	manifest := getManifest(&res.Data, ud)

	SendResponse(w, 200, manifest)
}

func getTemplateData() *configure.TemplateData {
	return &configure.TemplateData{
		Title:       "StremThru Wrap",
		Description: "Stremio Addon to Wrap another Addon with StremThru",
		Configs: []configure.Config{
			configure.Config{
				Key:         "manifest_url",
				Type:        "url",
				Default:     "",
				Title:       "Upstream Manifest URL",
				Description: "Manifest URL for the Upstream Addon",
				Required:    true,
			},
			configure.Config{
				Key:     "store",
				Type:    "select",
				Default: "",
				Title:   "Store Name",
				Options: []configure.ConfigOption{
					configure.ConfigOption{Value: "", Label: "StremThru"},
					configure.ConfigOption{Value: "alldebrid", Label: "AllDebrid"},
					configure.ConfigOption{Value: "debridlink", Label: "DebridLink"},
					configure.ConfigOption{Value: "offcloud", Label: "Offcloud"},
					configure.ConfigOption{Value: "premiumize", Label: "Premiumize"},
					configure.ConfigOption{Value: "realdebrid", Label: "RealDebrid"},
					configure.ConfigOption{Value: "torbox", Label: "TorBox"},
				},
				Required: false,
			},
			configure.Config{
				Key:         "token",
				Type:        "password",
				Default:     "",
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
		},
		Script: configure.GetScriptStoreTokenDescription("store", "token"),
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
		case "manifest_url":
			conf.Default = ud.ManifestURL
		case "store":
			conf.Default = ud.StoreName
		case "token":
			conf.Default = ud.StoreToken
		}
	}

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/manifest.json").String()
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

	var manifest_url_config *configure.Config
	var store_config *configure.Config
	var token_config *configure.Config
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "manifest_url":
			manifest_url_config = conf
		case "store":
			store_config = conf
		case "token":
			token_config = conf
		}
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			manifest_url_config.Error = uderr.manifestUrl
			store_config.Error = uderr.store
			token_config.Error = uderr.token
		} else {
			SendError(w, err)
			return
		}
	}

	if ctx.Store == nil {
		if ud.StoreName == "" {
			token_config.Error = "Invalid Token"
		} else {
			store_config.Error = "Invalid Store"
		}
	}

	if manifest_url_config.Error == "" {
		_, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl})
		if err != nil {
			manifest_url_config.Error = "Failed to fetch Manifest"
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

	url := ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/configure")
	q := url.Query()
	q.Set("try_install", "1")
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	resource := r.PathValue("resource")
	contentType := r.PathValue("contentType")
	id := r.PathValue("id")
	extra := r.PathValue("extra")

	if resource == string(stremio.ResourceNameStream) {
		res, err := addon.FetchStream(&stremio_addon.FetchStreamParams{
			BaseURL: ud.baseUrl,
			Type:    contentType,
			Id:      id,
			Extra:   extra,
		})
		if err != nil {
			SendError(w, err)
			return
		}

		ctx, err := ud.GetRequestContext(r)
		if err != nil {
			SendError(w, err)
			return
		}

		hashes := []string{}
		magnetByHash := map[string]core.MagnetLink{}
		for i := range res.Data.Streams {
			stream := &res.Data.Streams[i]
			if stream.URL == "" && stream.InfoHash != "" {
				magnet, err := core.ParseMagnetLink(stream.InfoHash)
				if err != nil {
					continue
				}
				hashes = append(hashes, magnet.Hash)
				magnetByHash[magnet.Hash] = magnet
			}
		}

		isCachedByHash := map[string]bool{}
		cmParams := &store.CheckMagnetParams{Magnets: hashes}
		cmParams.APIKey = ctx.StoreAuthToken
		cmRes, err := ctx.Store.CheckMagnet(cmParams)
		if err != nil {
			SendError(w, err)
			return
		}
		for _, item := range cmRes.Items {
			isCachedByHash[item.Hash] = item.Status == store.MagnetStatusCached
		}

		storeNamePrefix := "[" + strings.ToUpper(string(ctx.Store.GetName().Code())) + "] "
		for i := range res.Data.Streams {
			stream := &res.Data.Streams[i]
			if stream.URL == "" && stream.InfoHash != "" {
				magnet, ok := magnetByHash[strings.ToLower(stream.InfoHash)]
				if !ok {
					continue
				}
				stream.Name = storeNamePrefix + stream.Name
				if isCached, ok := isCachedByHash[magnet.Hash]; ok && isCached {
					stream.Name = "⚡ " + stream.Name
				}
				eud, err := ud.GetEncoded()
				if err != nil {
					continue
				}
				stream.URL = shared.ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/_/strem/" + magnet.Hash + "/" + strconv.Itoa(stream.FileIndex) + "/" + stream.BehaviorHints.Filename).String()
				stream.InfoHash = ""
				stream.FileIndex = 0
			} else if stream.URL != "" {
				if url, err := shared.CreateProxyLink(r, ctx, stream.URL); err == nil && url != stream.URL {
					stream.URL = url
					stream.Name = "✨ " + stream.Name
				}
			}
		}

		SendResponse(w, 200, res.Data)
		return
	}

	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.baseUrl,
		Resource: resource,
		Type:     contentType,
		Id:       id,
		Extra:    extra,
	})
}

func waitForMagnetStatus(ctx *context.RequestContext, m *store.GetMagnetData, status store.MagnetStatus, maxRetry int, retryInterval time.Duration) (*store.GetMagnetData, error) {
	retry := 0
	for m.Status != status && retry < maxRetry {
		gmParams := &store.GetMagnetParams{Id: m.Id}
		gmParams.APIKey = ctx.StoreAuthToken
		magnet, err := ctx.Store.GetMagnet(gmParams)
		if err != nil {
			return m, err
		}
		m = magnet
		time.Sleep(retryInterval)
		retry++
	}
	if m.Status != status {
		error := core.NewStoreError("torrent failed to reach status: " + string(status))
		error.StoreName = string(ctx.Store.GetName())
		return m, error
	}
	return m, nil
}

var stremLinkCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "stremio:wrap:streamLink",
	Lifetime: 3 * time.Hour,
})

func redirectToStaticVideo(w http.ResponseWriter, r *http.Request, cacheKey string, videoName string) {
	link := shared.ExtractRequestBaseURL(r).JoinPath("/v0/store/_/static/" + videoName + ".mp4").String()
	stremLinkCache.AddWithLifetime(cacheKey, link, 1*time.Minute)
	http.Redirect(w, r, link, http.StatusFound)
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	magnetHash := r.PathValue("magnetHash")
	fileName := r.PathValue("fileName")
	fileIdx := -1
	if idx, err := strconv.Atoi(r.PathValue("fileIdx")); err == nil {
		fileIdx = idx
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

	stremLink := ""

	cacheKey := strings.Join([]string{ctx.ClientIP, string(ctx.Store.GetName()), ctx.StoreAuthToken, magnetHash, strconv.Itoa(fileIdx), fileName}, ":")
	if stremLinkCache.Get(cacheKey, &stremLink) {
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	amParams := &store.AddMagnetParams{
		Magnet:   magnetHash,
		ClientIP: ctx.ClientIP,
	}
	amParams.APIKey = ctx.StoreAuthToken
	amRes, err := ctx.Store.AddMagnet(amParams)
	if err != nil {
		redirectToStaticVideo(w, r, cacheKey, "download_failed")
		return
	}

	magnet := &store.GetMagnetData{
		Id:      amRes.Id,
		Name:    amRes.Name,
		Hash:    amRes.Hash,
		Status:  amRes.Status,
		Files:   amRes.Files,
		AddedAt: amRes.AddedAt,
	}

	magnet, err = waitForMagnetStatus(ctx, magnet, store.MagnetStatusDownloaded, 12, 5*time.Second)
	if err != nil {
		if magnet.Status == store.MagnetStatusQueued || magnet.Status == store.MagnetStatusDownloading || magnet.Status == store.MagnetStatusProcessing {
			redirectToStaticVideo(w, r, cacheKey, "downloading")
			return
		}
		if magnet.Status == store.MagnetStatusFailed || magnet.Status == store.MagnetStatusInvalid || magnet.Status == store.MagnetStatusUnknown {
			redirectToStaticVideo(w, r, cacheKey, "download_failed")
			return
		}
		redirectToStaticVideo(w, r, cacheKey, "500")
		return
	}

	link := ""
	for i := range magnet.Files {
		f := &magnet.Files[i]
		if f.Name == fileName || (fileIdx != -1 && f.Idx == fileIdx) {
			link = f.Link
			break
		}
	}

	if link == "" {
		redirectToStaticVideo(w, r, cacheKey, "no_matching_file")
		return
	}

	glParams := &store.GenerateLinkParams{
		Link:     link,
		ClientIP: ctx.ClientIP,
	}
	glParams.APIKey = ctx.StoreAuthToken
	glRes, err := ctx.Store.GenerateLink(glParams)
	if err != nil {
		redirectToStaticVideo(w, r, cacheKey, "500")
		return
	}

	glRes, err = shared.GenerateStremThruLink(r, ctx, glRes.Link)
	if err != nil {
		redirectToStaticVideo(w, r, cacheKey, "500")
		return
	}

	stremLink = glRes.Link
	stremLinkCache.Add(cacheKey, stremLink)
	http.Redirect(w, r, stremLink, http.StatusFound)
}

func AddStremioWrapEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/stremio/wrap", handleRoot)
	mux.HandleFunc("/stremio/wrap/{$}", handleRoot)

	mux.HandleFunc("/stremio/wrap/manifest.json", handleManifest)
	mux.HandleFunc("/stremio/wrap/{userData}/manifest.json", handleManifest)

	mux.HandleFunc("/stremio/wrap/configure", handleConfigure)
	mux.HandleFunc("/stremio/wrap/{userData}/configure", handleConfigure)

	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}", handleResource)
	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}/{extra}", handleResource)

	mux.HandleFunc("/stremio/wrap/{userData}/_/strem/{magnetHash}/{fileIdx}/{$}", handleStrem)
	mux.HandleFunc("/stremio/wrap/{userData}/_/strem/{magnetHash}/{fileIdx}/{fileName}", handleStrem)
}
