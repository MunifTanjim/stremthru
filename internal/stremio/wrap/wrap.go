package stremio_wrap

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

var upstreamResolverCache = cache.NewCache[upstreamsResolver](&cache.CacheConfig{
	Name:     "stremio:wrap:upstreamResolver",
	Lifetime: 24 * time.Hour,
})

type upstreamsResolverEntry struct {
	Prefix  string
	Indices []int
}

type upstreamsResolver map[string][]upstreamsResolverEntry

func (usr upstreamsResolver) resolve(ud UserData, rName stremio.ResourceName, rType string, id string) []UserDataUpstream {
	upstreams := []UserDataUpstream{}
	key := string(rName) + ":" + rType
	if _, found := usr[key]; !found {
		return upstreams
	}
	for _, entry := range usr[key] {
		if strings.HasPrefix(id, entry.Prefix) {
			for _, idx := range entry.Indices {
				upstreams = append(upstreams, ud.Upstreams[idx])
			}
			break
		}
	}
	return upstreams
}

type UserDataUpstream struct {
	URL     string   `json:"u"`
	baseUrl *url.URL `json:"-"`
}

type UserData struct {
	ManifestURL string             `json:"manifest_url,omitempty"`
	Upstreams   []UserDataUpstream `json:"upstreams"`
	StoreName   string             `json:"store"`
	StoreToken  string             `json:"token"`
	CachedOnly  bool               `json:"cached,omitempty"`

	encoded   string             `json:"-"`
	manifests []stremio.Manifest `json:"-"`
	resolver  upstreamsResolver  `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	if len(ud.Upstreams) == 0 {
		return false
	}
	for i := range ud.Upstreams {
		if ud.Upstreams[i].URL == "" {
			return false
		}
	}
	return ud.StoreToken != ""
}

func (ud UserData) GetEncoded(forceRefresh bool) (string, error) {
	if ud.encoded == "" || forceRefresh {
		blob, err := json.Marshal(ud)
		if err != nil {
			return "", err
		}
		ud.encoded = core.Base64Encode(string(blob))
	}

	return ud.encoded, nil
}

type userDataError struct {
	upstreamUrl []string
	store       string
	token       string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	for _, err := range uderr.upstreamUrl {
		if err != "" {
			str.WriteString("upstream_url: ")
			str.WriteString(err)
			hasSome = true
		}
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

	upstreamUrlErrors := []string{}
	hasUpstreamUrlErrors := false
	for i := range ud.Upstreams {
		up := &ud.Upstreams[i]
		if up.baseUrl == nil {
			upstreamUrlErrors = append(upstreamUrlErrors, "Invalid Manifest URL("+up.URL+")")
			hasUpstreamUrlErrors = true
		} else {
			upstreamUrlErrors = append(upstreamUrlErrors, "")
		}
	}
	if hasUpstreamUrlErrors {
		return ctx, &userDataError{upstreamUrl: upstreamUrlErrors}
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

func (ud UserData) getUpstreamManifests(ctx *context.RequestContext) ([]stremio.Manifest, []error) {
	if ud.manifests == nil {
		var wg sync.WaitGroup

		manifests := make([]stremio.Manifest, len(ud.Upstreams))
		errs := make([]error, len(ud.Upstreams))
		hasError := false
		for i := range ud.Upstreams {
			up := &ud.Upstreams[i]
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: up.baseUrl, ClientIP: ctx.ClientIP})
				manifests[i] = res.Data
				errs[i] = err
				if err != nil {
					hasError = true
				}
			}()
		}

		wg.Wait()

		if hasError {
			return manifests, errs
		}

		ud.manifests = manifests
	}

	return ud.manifests, nil
}

func (ud UserData) getUpstreamsResolver(ctx *context.RequestContext) (upstreamsResolver, error) {
	eud, err := ud.GetEncoded(false)
	if err != nil {
		return nil, err
	}
	if ud.resolver == nil {
		if upstreamResolverCache.Get(eud, &ud.resolver) {
			return ud.resolver, nil
		}

		manifests, errs := ud.getUpstreamManifests(ctx)
		if errs != nil {
			return nil, errors.Join(errs...)
		}

		resolver := upstreamsResolver{}
		entryIdxMap := map[string]int{}
		for mIdx := range manifests {
			m := &manifests[mIdx]
			for _, r := range m.Resources {
				if r.Name == stremio.ResourceNameAddonCatalog || r.Name == stremio.ResourceNameCatalog {
					continue
				}

				idPrefixes := getManifestResourceIdPrefixes(m, r)
				for _, rType := range getManifestResourceTypes(m, r) {
					key := string(r.Name) + ":" + string(rType)
					if _, found := resolver[key]; !found {
						resolver[key] = []upstreamsResolverEntry{}
					}
					for _, idPrefix := range idPrefixes {
						idPrefixKey := key + ":" + idPrefix
						if idx, found := entryIdxMap[idPrefixKey]; found {
							resolver[key][idx].Indices = append(resolver[key][idx].Indices, mIdx)
						} else {
							resolver[key] = append(resolver[key], upstreamsResolverEntry{
								Prefix:  idPrefix,
								Indices: []int{mIdx},
							})
							entryIdxMap[idPrefixKey] = len(resolver[key]) - 1
						}
					}
				}
			}
		}

		err = upstreamResolverCache.Add(eud, resolver)
		if err != nil {
			return nil, err
		}

		ud.resolver = resolver
	}

	return ud.resolver, nil
}

func (ud UserData) getUpstreams(ctx *context.RequestContext, rName stremio.ResourceName, rType, id string) ([]UserDataUpstream, error) {
	switch rName {
	case stremio.ResourceNameAddonCatalog:
		return []UserDataUpstream{}, nil
	case stremio.ResourceNameCatalog:
		return []UserDataUpstream{}, nil
	default:
		if len(ud.Upstreams) == 1 {
			return ud.Upstreams, nil
		}

		resolver, err := ud.getUpstreamsResolver(ctx)
		if err != nil {
			return nil, err
		}
		return resolver.resolve(ud, rName, rType, id), nil
	}
}

func parseCatalogId(id string, ud *UserData) (idx int, catalogId string, err error) {
	idxStr, catalogId, ok := strings.Cut(id, "::")
	if !ok {
		return -1, "", errors.New("invalid id")
	}
	idx, err = strconv.Atoi(idxStr)
	if err != nil {
		return -1, "", err
	}
	if len(ud.Upstreams) <= idx {
		return -1, "", errors.New("invalid id")
	}
	return idx, catalogId, nil
}

func (ud UserData) fetchAddonCatalog(ctx *context.RequestContext, w http.ResponseWriter, r *http.Request, rType, id string) {
	idx, catalogId, err := parseCatalogId(id, &ud)
	if err != nil {
		SendError(w, err)
		return
	}
	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.Upstreams[idx].baseUrl,
		Resource: string(stremio.ResourceNameAddonCatalog),
		Type:     rType,
		Id:       catalogId,
		ClientIP: ctx.ClientIP,
	})
}

func (ud UserData) fetchCatalog(ctx *context.RequestContext, w http.ResponseWriter, r *http.Request, rType, id, extra string) {
	idx, catalogId, err := parseCatalogId(id, &ud)
	if err != nil {
		SendError(w, err)
		return
	}
	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.Upstreams[idx].baseUrl,
		Resource: string(stremio.ResourceNameCatalog),
		Type:     rType,
		Id:       catalogId,
		Extra:    extra,
		ClientIP: ctx.ClientIP,
	})
}

func (ud UserData) fetchMeta(ctx *context.RequestContext, w http.ResponseWriter, r *http.Request, rType, id, extra string) error {
	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameMeta, rType, id)
	if err != nil {
		return err
	}

	upstream := upstreams[0]

	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  upstream.baseUrl,
		Resource: string(stremio.ResourceNameMeta),
		Type:     rType,
		Id:       id,
		Extra:    extra,
		ClientIP: ctx.ClientIP,
	})
	return nil
}

func (ud UserData) fetchStream(ctx *context.RequestContext, r *http.Request, rType, id string) (*stremio.StreamHandlerResponse, error) {
	eud, err := ud.GetEncoded(false)
	if err != nil {
		return nil, err
	}

	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameStream, rType, id)
	if err != nil {
		return nil, err
	}

	chunks := make([][]stremio.Stream, len(upstreams))
	errs := make([]error, len(upstreams))

	var wg sync.WaitGroup
	for i := range upstreams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := addon.FetchStream(&stremio_addon.FetchStreamParams{
				BaseURL:  upstreams[i].baseUrl,
				Type:     rType,
				Id:       id,
				ClientIP: ctx.ClientIP,
			})
			chunks[i] = res.Data.Streams
			errs[i] = err
		}()
	}
	wg.Wait()

	allStreams := []stremio.Stream{}
	for i := range chunks {
		if errs[i] != nil {
			log.Println("[stremio/wrap] failed to fetch streams", errs[i])
			continue
		}
		allStreams = append(allStreams, chunks[i]...)
	}

	hashes := []string{}
	magnetByHash := map[string]core.MagnetLink{}
	for i := range allStreams {
		stream := &allStreams[i]
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
			return nil, err
		}
		for _, item := range cmRes.Items {
			isCachedByHash[item.Hash] = item.Status == store.MagnetStatusCached
		}

		storeNamePrefix = "[" + strings.ToUpper(string(ctx.Store.GetName().Code())) + "] "
	}

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}
	for i := range allStreams {
		stream := &allStreams[i]
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

	return &stremio.StreamHandlerResponse{
		Streams: streams,
	}, nil
}

func (ud UserData) fetchSubtitles(ctx *context.RequestContext, rType, id, extra string) (*stremio.SubtitlesHandlerResponse, error) {
	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameSubtitles, rType, id)
	if err != nil {
		return nil, err
	}

	chunks := make([][]stremio.Subtitle, len(upstreams))
	errs := make([]error, len(upstreams))

	var wg sync.WaitGroup
	for i := range upstreams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := addon.FetchSubtitles(&stremio_addon.FetchSubtitlesParams{
				BaseURL:  upstreams[i].baseUrl,
				Type:     rType,
				Id:       id,
				Extra:    extra,
				ClientIP: ctx.ClientIP,
			})
			chunks[i] = res.Data.Subtitles
			errs[i] = err
		}()
	}
	wg.Wait()

	subtitles := []stremio.Subtitle{}
	for i := range chunks {
		if errs[i] != nil {
			log.Println("[stremio/wrap] failed to fetch subtitles", errs[i])
			continue
		}
		subtitles = append(subtitles, chunks[i]...)
	}

	return &stremio.SubtitlesHandlerResponse{
		Subtitles: subtitles,
	}, nil
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

		if data.ManifestURL != "" {
			data.Upstreams = []UserDataUpstream{
				{
					URL: data.ManifestURL,
				},
			}
			data.ManifestURL = ""
			_, err := data.GetEncoded(true)
			if err != nil {
				return nil, err
			}
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		upstreams_length := 1
		if v := r.Form.Get("upstreams_length"); v != "" {
			upstreams_length, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		}

		for idx := range upstreams_length {
			upURL := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].url")
			if strings.HasPrefix(upURL, "stremio:") {
				upURL = "https:" + strings.TrimPrefix(upURL, "stremio:")
			}
			if upURL != "" {
				data.Upstreams = append(data.Upstreams, UserDataUpstream{
					URL: upURL,
				})
			}
		}

		data.StoreName = r.Form.Get("store")
		data.StoreToken = r.Form.Get("token")
		data.CachedOnly = r.Form.Get("cached") == "on"

		_, err = data.GetEncoded(false)
		if err != nil {
			return nil, err
		}
	}

	for i := range data.Upstreams {
		up := &data.Upstreams[i]
		if up.URL != "" {
			if baseUrl, err := url.Parse(up.URL); err == nil {
				if strings.HasSuffix(baseUrl.Path, "/manifest.json") {
					baseUrl.Path = strings.TrimSuffix(baseUrl.Path, "/manifest.json")
					up.baseUrl = baseUrl
				}
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

	manifests, errs := ud.getUpstreamManifests(ctx)
	if errs != nil {
		serr := shared.ErrorInternalServerError(r, "failed to fetch upstream manifests")
		serr.Cause = errors.Join(errs...)
		serr.Send(w)
		return
	}

	manifest := getManifest(manifests, ud)

	SendResponse(w, 200, manifest)
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

	td := getTemplateData(ud)
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

	if action := r.Header.Get("x-addon-configure-action"); action != "" {
		switch action {
		case "add-upstream":
			td.Upstreams = append(td.Upstreams, UpstreamAddon{
				URL: "",
			})
		case "remove-upstream":
			end := len(td.Upstreams) - 1
			if end == 0 {
				end = 1
			}
			td.Upstreams = append([]UpstreamAddon{}, td.Upstreams[0:end]...)
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, err)
			return
		}
		SendHTML(w, 200, page)
		return
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
				for i, err := range uderr.upstreamUrl {
					td.Upstreams[i].Error = err
				}
				store_config.Error = uderr.store
				token_config.Error = uderr.token
			} else {
				SendError(w, err)
				return
			}
		}

		manifests, errs := ud.getUpstreamManifests(ctx)
		for i := range manifests {
			tup := &td.Upstreams[i]
			manifest := manifests[i]

			if tup.Error == "" {
				if errs != nil && errs[i] != nil {
					core.LogError("[stremio/wrap] failed to fetch manifest", errs[i])
					tup.Error = "Failed to fetch Manifest"
					continue
				}

				if manifest.BehaviorHints != nil && manifest.BehaviorHints.Configurable {
					tup.IsConfigurable = true
				}
			}
		}

		if !td.HasUpstreamError() {
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
			if eud, err := ud.GetEncoded(false); err == nil {
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

	eud, err := ud.GetEncoded(true)
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

	switch stremio.ResourceName(resource) {
	case stremio.ResourceNameAddonCatalog:
		ud.fetchAddonCatalog(ctx, w, r, contentType, id)
	case stremio.ResourceNameCatalog:
		ud.fetchCatalog(ctx, w, r, contentType, id, extra)
	case stremio.ResourceNameMeta:
		err = ud.fetchMeta(ctx, w, r, contentType, id, extra)
		if err != nil {
			SendError(w, err)
		}
		return
	case stremio.ResourceNameStream:
		res, err := ud.fetchStream(ctx, r, contentType, id)
		if err != nil {
			SendError(w, err)
			return
		}
		SendResponse(w, 200, res)
		return

	case stremio.ResourceNameSubtitles:
		res, err := ud.fetchSubtitles(ctx, contentType, id, extra)
		if err != nil {
			SendError(w, err)
			return
		}
		SendResponse(w, 200, res)
		return
	default:
		addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
			BaseURL:  ud.Upstreams[0].baseUrl,
			Resource: resource,
			Type:     contentType,
			Id:       id,
			Extra:    extra,
			ClientIP: ctx.ClientIP,
		})
	}
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
