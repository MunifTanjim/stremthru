package stremio_wrap

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/store/video"
	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"golang.org/x/sync/singleflight"
)

var addon = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

type UserData struct {
	ManifestURL string   `json:"manifest_url"`
	StoreName   string   `json:"store"`
	StoreToken  string   `json:"token"`
	CachedOnly  bool     `json:"cached"`
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
		if err != nil {
			return nil, err
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		data.ManifestURL = r.Form.Get("manifest_url")
		if strings.HasPrefix(data.ManifestURL, "stremio:") {
			data.ManifestURL = "https:" + strings.TrimPrefix(data.ManifestURL, "stremio:")
		}

		data.StoreName = r.Form.Get("store")
		data.StoreToken = r.Form.Get("token")
		data.CachedOnly = r.Form.Get("cached") == "on"
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	if data.ManifestURL != "" {
		if baseUrl, err := url.Parse(data.ManifestURL); err == nil {
			if strings.HasSuffix(baseUrl.Path, "/manifest.json") {
				baseUrl.Path = strings.TrimSuffix(baseUrl.Path, "/manifest.json")
				data.baseUrl = baseUrl
			}
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

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		SendError(w, err)
		return
	}

	res, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl, ClientIP: ctx.ClientIP})
	if err != nil {
		SendError(w, err)
		return
	}

	manifest := getManifest(&res.Data, ud)

	SendResponse(w, 200, manifest)
}

func getStoreNameConfig() configure.Config {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "alldebrid", Label: "AllDebrid"},
		{Value: "debridlink", Label: "DebridLink"},
		{Value: "easydebrid", Label: "EasyDebrid"},
		{Value: "offcloud", Label: "Offcloud"},
		{Value: "pikpak", Label: "PikPak"},
		{Value: "premiumize", Label: "Premiumize"},
		{Value: "realdebrid", Label: "RealDebrid"},
		{Value: "torbox", Label: "TorBox"},
	}
	if !config.ProxyStreamEnabled {
		options[0].Disabled = true
		options[0].Label = ""
	}
	config := configure.Config{
		Key:      "store",
		Type:     "select",
		Default:  "",
		Title:    "Store Name",
		Options:  options,
		Required: !config.ProxyStreamEnabled,
	}
	return config
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
	td.Upstream.URL = ud.ManifestURL
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "store":
			conf.Default = ud.StoreName
		case "token":
			conf.Default = ud.StoreToken
		case "cached":
			if ud.CachedOnly {
				conf.Default = "checked"
			}
		}
	}

	if ud.encoded != "" {
		var store_config *configure.Config
		var token_config *configure.Config
		for i := range td.Configs {
			conf := &td.Configs[i]
			switch conf.Key {
			case "store":
				store_config = conf
			case "token":
				token_config = conf
			}
		}

		ctx, err := ud.GetRequestContext(r)
		if err != nil {
			if uderr, ok := err.(*userDataError); ok {
				td.HasError.Upstream = true
				td.Message.Upstream = uderr.manifestUrl
				store_config.Error = uderr.store
				token_config.Error = uderr.token
			} else {
				SendError(w, err)
				return
			}
		}

		if !td.HasError.Upstream {
			manifest, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl, ClientIP: ctx.ClientIP})
			if err != nil {
				core.LogError("[stremio/wrap] failed to fetch manifest", err)
				td.HasError.Upstream = true
				td.Message.Upstream = "Failed to fetch Manifest"
			} else if manifest.Data.BehaviorHints != nil && manifest.Data.BehaviorHints.Configurable {
				td.Upstream.IsConfigurable = true
			}
		}

		if !td.HasError.Upstream {
			if ctx.Store == nil {
				if ud.StoreName == "" {
					token_config.Error = "Invalid Token"
				} else {
					store_config.Error = "Invalid Store"
				}
			} else {
				params := &store.GetUserParams{}
				params.APIKey = ctx.StoreAuthToken
				_, err := ctx.Store.GetUser(params)
				if err != nil {
					core.LogError("[stremio/wrap] failed to access store", err)
					token_config.Error = "Failed to access store"
				}
			}
		}
	}

	hasError := td.HasFieldError()

	if IsMethod(r, http.MethodGet) || hasError {
		if !hasError && ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/manifest.json").String()
			}
		}

		page, err := getPage(td)
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

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		SendError(w, err)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, err)
		return
	}

	if resource == string(stremio.ResourceNameStream) {
		res, err := addon.FetchStream(&stremio_addon.FetchStreamParams{
			BaseURL:  ud.baseUrl,
			Type:     contentType,
			Id:       id,
			Extra:    extra,
			ClientIP: ctx.ClientIP,
		})
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

		stremId := strings.TrimSuffix(id, ".json")

		storeNamePrefix := ""
		isCachedByHash := map[string]bool{}
		if len(hashes) > 0 {
			cmParams := &store.CheckMagnetParams{Magnets: hashes}
			cmParams.APIKey = ctx.StoreAuthToken
			cmParams.ClientIP = ctx.ClientIP
			cmParams.SId = stremId
			cmRes, err := ctx.Store.CheckMagnet(cmParams)
			if err != nil {
				SendError(w, err)
				return
			}
			for _, item := range cmRes.Items {
				isCachedByHash[item.Hash] = item.Status == store.MagnetStatusCached
			}

			storeNamePrefix = "[" + strings.ToUpper(string(ctx.Store.GetName().Code())) + "] "
		}

		cachedStreams := []stremio.Stream{}
		uncachedStreams := []stremio.Stream{}
		for i := range res.Data.Streams {
			stream := &res.Data.Streams[i]
			if stream.URL == "" && stream.InfoHash != "" {
				magnet, ok := magnetByHash[strings.ToLower(stream.InfoHash)]
				if !ok {
					continue
				}
				stream.Name = storeNamePrefix + stream.Name
				url := shared.ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/_/strem/" + magnet.Hash + "/" + strconv.Itoa(stream.FileIndex) + "/")
				if stream.BehaviorHints != nil && stream.BehaviorHints.Filename != "" {
					url = url.JoinPath(stream.BehaviorHints.Filename)
				}
				url.RawQuery = "sid=" + stremId
				stream.URL = url.String()
				stream.InfoHash = ""
				stream.FileIndex = 0

				if isCached, ok := isCachedByHash[magnet.Hash]; ok && isCached {
					stream.Name = "⚡ " + stream.Name
					cachedStreams = append(cachedStreams, *stream)
				} else if !ud.CachedOnly {
					uncachedStreams = append(uncachedStreams, *stream)
				}
			} else if stream.URL != "" {
				var headers map[string]string
				if stream.BehaviorHints != nil && stream.BehaviorHints.ProxyHeaders != nil && stream.BehaviorHints.ProxyHeaders.Request != nil {
					headers = stream.BehaviorHints.ProxyHeaders.Request
				}

				if url, err := shared.CreateProxyLink(r, ctx, stream.URL, headers, config.TUNNEL_TYPE_AUTO); err == nil && url != stream.URL {
					stream.URL = url
					stream.Name = "✨ " + stream.Name
				}
				cachedStreams = append(cachedStreams, *stream)
			}
		}

		streams := make([]stremio.Stream, len(cachedStreams)+len(uncachedStreams))
		idx := 0
		for i := range cachedStreams {
			streams[idx] = cachedStreams[i]
			idx++
		}
		for i := range uncachedStreams {
			streams[idx] = uncachedStreams[i]
			idx++
		}

		res.Data.Streams = streams

		SendResponse(w, 200, res.Data)
		return
	}

	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.baseUrl,
		Resource: resource,
		Type:     contentType,
		Id:       id,
		Extra:    extra,
		ClientIP: ctx.ClientIP,
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
	url := store_video.Redirect(videoName, w, r)
	stremLinkCache.AddWithLifetime(cacheKey, url, 1*time.Minute)
}

var stremGroup singleflight.Group

type stremResult struct {
	link        string
	error_log   string
	error_video string
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
		if err != nil {
			core.LogError("[stremio/wrap] failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "").Send(w)
		return
	}

	cacheKey := strings.Join([]string{ctx.ClientIP, string(ctx.Store.GetName()), ctx.StoreAuthToken, magnetHash, strconv.Itoa(fileIdx), fileName}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (interface{}, error) {
		amParams := &store.AddMagnetParams{
			Magnet:   magnetHash,
			ClientIP: ctx.ClientIP,
		}
		amParams.APIKey = ctx.StoreAuthToken
		amRes, err := ctx.Store.AddMagnet(amParams)
		if err != nil {
			return &stremResult{
				error_log:   "failed to add magnet",
				error_video: "download_failed",
			}, err
		}

		magnet := &store.GetMagnetData{
			Id:      amRes.Id,
			Name:    amRes.Name,
			Hash:    amRes.Hash,
			Status:  amRes.Status,
			Files:   amRes.Files,
			AddedAt: amRes.AddedAt,
		}

		magnet, err = waitForMagnetStatus(ctx, magnet, store.MagnetStatusDownloaded, 3, 5*time.Second)

		query := r.URL.Query()
		sid := query.Get("sid")
		if sid == "" {
			sid = "*"
		}

		go buddy.TrackMagnet(ctx.Store, magnet.Hash, magnet.Files, sid, magnet.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)

		if err != nil {
			strem := &stremResult{
				error_log:   "failed wait for magnet status",
				error_video: "500",
			}
			if magnet.Status == store.MagnetStatusQueued || magnet.Status == store.MagnetStatusDownloading || magnet.Status == store.MagnetStatusProcessing {
				strem.error_video = "downloading"
			} else if magnet.Status == store.MagnetStatusFailed || magnet.Status == store.MagnetStatusInvalid || magnet.Status == store.MagnetStatusUnknown {
				strem.error_video = "download_failed"
			}
			return strem, err
		}

		var file *store.MagnetFile
		if fileName != "" {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if f.Name == fileName {
					file = f
					break
				}
			}
		}
		if file == nil && fileIdx != -1 {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if f.Idx == fileIdx {
					file = f
					break
				}
			}
		}
		if file == nil {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if file == nil || file.Size < f.Size {
					file = f
				}
			}
		}

		link := ""
		if file != nil {
			link = file.Link
		}
		if link == "" {
			return &stremResult{
				error_log:   "no matching file found for (" + sid + " - " + magnet.Hash + ")",
				error_video: "no_matching_file",
			}, nil
		}

		glRes, err := shared.GenerateStremThruLink(r, ctx, link)
		if err != nil {
			return &stremResult{
				error_log:   "failed to generate stremthru link",
				error_video: "500",
			}, err
		}

		stremLinkCache.Add(cacheKey, glRes.Link)

		return &stremResult{
			link: glRes.Link,
		}, nil
	})

	strem := result.(*stremResult)

	if strem.error_log != "" {
		if err != nil {
			core.LogError("[stremio/wrap] "+strem.error_log, err)
		} else {
			log.Println("[stremio/wrap] " + strem.error_log)
		}
		redirectToStaticVideo(w, r, cacheKey, strem.error_video)
		return
	}

	http.Redirect(w, r, strem.link, http.StatusFound)
}

func AddStremioWrapEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	mux.HandleFunc("/stremio/wrap", handleRoot)
	mux.HandleFunc("/stremio/wrap/{$}", handleRoot)

	mux.HandleFunc("/stremio/wrap/manifest.json", withCors(handleManifest))
	mux.HandleFunc("/stremio/wrap/{userData}/manifest.json", withCors(handleManifest))

	mux.HandleFunc("/stremio/wrap/configure", handleConfigure)
	mux.HandleFunc("/stremio/wrap/{userData}/configure", handleConfigure)

	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}", withCors(handleResource))
	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}/{extra}", withCors(handleResource))

	mux.HandleFunc("/stremio/wrap/{userData}/_/strem/{magnetHash}/{fileIdx}/{$}", withCors(handleStrem))
	mux.HandleFunc("/stremio/wrap/{userData}/_/strem/{magnetHash}/{fileIdx}/{fileName}", withCors(handleStrem))
}
