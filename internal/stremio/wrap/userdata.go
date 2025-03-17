package stremio_wrap

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

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
	URL              string                         `json:"u"`
	baseUrl          *url.URL                       `json:"-"`
	ExtractorId      string                         `json:"e,omitempty"`
	extractor        StreamTransformerExtractorBlob `json:"-"`
	NoContentProxy   bool                           `json:"ncp,omitempty"`
	ReconfigureStore bool                           `json:"rs,omitempty"`
}

type UserDataStoreCode string

func (udsc UserDataStoreCode) IsStremThru() bool {
	return !IsPublicInstance && udsc == ""
}

func (udsc UserDataStoreCode) GetCodes(user string) []string {
	codes := []string{}

	storeNames := config.StoreAuthToken.ListStores(user)
	for _, name := range storeNames {
		codes = append(codes, string(store.StoreName(name).Code()))
	}

	return codes
}

type UserDataStore struct {
	Code  UserDataStoreCode `json:"c"`
	Token string            `json:"t"`
}

type UserData struct {
	Upstreams   []UserDataUpstream `json:"upstreams"`
	ManifestURL string             `json:"manifest_url,omitempty"`

	Stores     []UserDataStore `json:"stores"`
	StoreName  string          `json:"store,omitempty"`
	StoreToken string          `json:"token,omitempty"`

	CachedOnly bool `json:"cached,omitempty"`

	TemplateId string                        `json:"template,omitempty"`
	template   StreamTransformerTemplateBlob `json:"-"`

	Sort string `json:"sort,omitempty"`

	encoded          string             `json:"-"` // correctly configured
	manifests        []stremio.Manifest `json:"-"`
	resolver         upstreamsResolver  `json:"-"`
	stores           multiStore         `json:"-"`
	isStremThruStore bool               `json:"-"`
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "wrap",
})

func (ud UserData) HasRequiredValues() bool {
	if len(ud.Upstreams) == 0 {
		return false
	}
	for i := range ud.Upstreams {
		if ud.Upstreams[i].URL == "" {
			return false
		}
	}
	if len(ud.Stores) == 0 {
		return false
	}
	for i := range ud.Stores {
		s := &ud.Stores[i]
		if s.Code.IsStremThru() && len(ud.Stores) > 1 {
			return false
		}
		if s.Token == "" {
			return false
		}
	}
	return true
}

func (ud *UserData) GetEncoded() string {
	return ud.encoded
}

func (ud *UserData) SetEncoded(encoded string) {
	ud.encoded = encoded
}

func (ud *UserData) Ptr() *UserData {
	return ud
}

type userDataError struct {
	upstreamUrl []string
	store       []string
	token       []string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	for i, err := range uderr.upstreamUrl {
		if err != "" {
			if hasSome {
				str.WriteString(", ")
				hasSome = false
			}

			str.WriteString("upstream_url[" + strconv.Itoa(i) + "]: ")
			str.WriteString(err)
			hasSome = true
		}
	}
	for i, err := range uderr.store {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("store[" + strconv.Itoa(i) + "]: ")
		str.WriteString(err)
		hasSome = true
	}
	for i, err := range uderr.token {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("token[" + strconv.Itoa(i) + "]: ")
		str.WriteString(err)
		hasSome = true

	}
	return str.String()
}

func (ud *UserData) GetRequestContext(r *http.Request) (*context.StoreContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &context.StoreContext{
		Log: rCtx.Log,
	}

	upstreamUrlErrors := []string{}
	hasUpstreamUrlErrors := false
	for i := range ud.Upstreams {
		up := &ud.Upstreams[i]
		if up.baseUrl == nil {
			upstreamUrlErrors = append(upstreamUrlErrors, "Invalid Manifest URL")
			hasUpstreamUrlErrors = true
		} else {
			upstreamUrlErrors = append(upstreamUrlErrors, "")
		}
	}
	if hasUpstreamUrlErrors {
		return ctx, &userDataError{upstreamUrl: upstreamUrlErrors}
	}

	if len(ud.Stores) == 1 && ud.Stores[0].Code.IsStremThru() {
		token := ud.Stores[0].Token
		auth, err := core.ParseBasicAuth(token)
		if err != nil {
			return ctx, &userDataError{token: []string{err.Error()}}
		}
		password := config.ProxyAuthPassword.GetPassword(auth.Username)
		if password == "" || password != auth.Password {
			return ctx, &userDataError{token: []string{"invalid token"}}
		} else {
			ctx.IsProxyAuthorized = true
			ctx.ProxyAuthUser = auth.Username
			ctx.ProxyAuthPassword = auth.Password
		}

		storeNames := config.StoreAuthToken.ListStores(auth.Username)
		stores := make(multiStore, len(storeNames))
		for i, storeName := range storeNames {
			stores[i] = resolvedStore{
				store:     shared.GetStore(storeName),
				authToken: config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, storeName),
			}
		}
		ud.stores = stores
		ud.isStremThruStore = true
	} else {
		stores := make(multiStore, len(ud.Stores))
		for i := range ud.Stores {
			s := &ud.Stores[i]
			stores[i] = resolvedStore{
				store:     shared.GetStore(string(store.StoreCode(s.Code).Name())),
				authToken: s.Token,
			}
		}
		ud.stores = stores
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
}

func (ud UserData) getUpstreamManifests(ctx *context.StoreContext) ([]stremio.Manifest, []error) {
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

func (ud UserData) getUpstreamsResolver(ctx *context.StoreContext) (upstreamsResolver, error) {
	eud := ud.GetEncoded()

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

		err := upstreamResolverCache.Add(eud, resolver)
		if err != nil {
			return nil, err
		}

		ud.resolver = resolver
	}

	return ud.resolver, nil
}

func (ud UserData) getUpstreams(ctx *context.StoreContext, rName stremio.ResourceName, rType, id string) ([]UserDataUpstream, error) {
	switch rName {
	case stremio.ResourceNameAddonCatalog:
		return []UserDataUpstream{}, nil
	case stremio.ResourceNameCatalog:
		return []UserDataUpstream{}, nil
	default:
		upstreamsCount := len(ud.Upstreams)
		if upstreamsCount == 1 {
			return ud.Upstreams, nil
		}

		if IsPublicInstance {
			if rName == stremio.ResourceNameMeta || rName == stremio.ResourceNameSubtitles {
				if upstreamsCount > 1 {
					return []UserDataUpstream{}, nil
				}
			}
			return ud.Upstreams, nil
		}

		resolver, err := ud.getUpstreamsResolver(ctx)
		if err != nil {
			return nil, err
		}
		return resolver.resolve(ud, rName, rType, id), nil
	}
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}
	data.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(data); err != nil {
			return nil, err
		}
		if data.encoded == "" {
			return data, nil
		}

		shouldResync := false
		if data.StoreToken != "" {
			data.Stores = []UserDataStore{
				{
					Code:  UserDataStoreCode(store.StoreName(data.StoreName).Code()),
					Token: data.StoreToken,
				},
			}
			data.StoreName = ""
			data.StoreToken = ""
			shouldResync = true
		}

		if data.ManifestURL != "" {
			data.Upstreams = []UserDataUpstream{
				{
					URL: data.ManifestURL,
				},
			}
			data.ManifestURL = ""
			shouldResync = true
		}

		if shouldResync {
			if err := udManager.Sync(data); err != nil {
				return nil, err
			}
		}

		hasMissingExtractor := false
		for i := range data.Upstreams {
			up := &data.Upstreams[i]

			if up.ExtractorId != "" {
				if config.IsPublicInstance {
					up.ExtractorId = getNewTransformerExtractorId(up.ExtractorId)
				}

				if extractor, err := getExtractor(up.ExtractorId); err != nil {
					LogError(r, fmt.Sprintf("failed to fetch extractor(%s)", up.ExtractorId), err)
					hasMissingExtractor = true
				} else {
					up.extractor = extractor
				}
			} else {
				hasMissingExtractor = true
			}
		}

		if !hasMissingExtractor && data.TemplateId != "" {
			if config.IsPublicInstance && !strings.HasPrefix(data.TemplateId, BUILTIN_TRANSFORMER_ENTITY_ID_PREFIX) {
				data.TemplateId = BUILTIN_TRANSFORMER_ENTITY_ID_PREFIX + data.TemplateId
			}

			if template, err := getTemplate(data.TemplateId); err != nil {
				LogError(r, fmt.Sprintf("failed to fetch template(%s)", data.TemplateId), err)
			} else {
				data.template = template
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

		data.Sort = r.Form.Get("sort")

		data.TemplateId = r.Form.Get("transformer.template_id")
		data.template = StreamTransformerTemplateBlob{
			Name:        r.Form.Get("transformer.template.name"),
			Description: r.Form.Get("transformer.template.description"),
		}

		for idx := range upstreams_length {
			upURL := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].url")
			if strings.HasPrefix(upURL, "stremio:") {
				upURL = "https:" + strings.TrimPrefix(upURL, "stremio:")
			}
			if strings.HasSuffix(upURL, "/configure") {
				upURL = strings.TrimSuffix(upURL, "/configure") + "/manifest.json"
			}
			extractorId := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].transformer.extractor_id")
			up := UserDataUpstream{
				URL:         upURL,
				ExtractorId: extractorId,
			}
			extractor := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].transformer.extractor")
			if extractor != "" {
				up.extractor = StreamTransformerExtractorBlob(extractor)
			}
			if upURL != "" || extractorId != "" || extractor != "" {
				up.NoContentProxy = r.Form.Get("upstreams["+strconv.Itoa(idx)+"].no_content_proxy") == "on"
				up.ReconfigureStore = r.Form.Get("upstreams["+strconv.Itoa(idx)+"].reconfigure_store") == "on"
				data.Upstreams = append(data.Upstreams, up)
			}
		}

		stores_length := 1
		if v := r.Form.Get("stores_length"); v != "" {
			stores_length, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		}

		for idx := range stores_length {
			code := r.Form.Get("stores[" + strconv.Itoa(idx) + "].code")
			token := r.Form.Get("stores[" + strconv.Itoa(idx) + "].token")
			if code == "" {
				data.Stores = []UserDataStore{
					{
						Code:  UserDataStoreCode(code),
						Token: token,
					},
				}
				break
			} else {
				data.Stores = append(data.Stores, UserDataStore{
					Code:  UserDataStoreCode(code),
					Token: token,
				})
			}
		}

		data.CachedOnly = r.Form.Get("cached") == "on"

		isStoreStremThru := false
		for i := range data.Stores {
			if data.Stores[i].Code.IsStremThru() {
				isStoreStremThru = true
				break
			}
		}

		if !isStoreStremThru {
			for i := range data.Upstreams {
				up := &data.Upstreams[i]
				up.NoContentProxy = false
			}
		}
	}

	if IsPublicInstance && len(data.Upstreams) > MaxPublicInstanceUpstreamCount {
		data.Upstreams = data.Upstreams[0:MaxPublicInstanceUpstreamCount]
	}

	for i := range data.Upstreams {
		up := &data.Upstreams[i]
		if up.URL != "" {
			if baseUrl, err := url.Parse(up.URL); err == nil {
				if strings.HasSuffix(baseUrl.Path, "/manifest.json") {
					if baseUrl.RawPath != "" && baseUrl.RawPath != baseUrl.Path {
						baseUrl.RawPath = strings.TrimSuffix(baseUrl.RawPath, "/manifest.json")
						if baseUrl.Path, err = url.PathUnescape(baseUrl.RawPath); err != nil {
							return nil, err
						}
					} else {
						baseUrl.Path = strings.TrimSuffix(baseUrl.Path, "/manifest.json")
					}
					up.baseUrl = baseUrl
				}
			}
		}
	}

	return data, nil
}
