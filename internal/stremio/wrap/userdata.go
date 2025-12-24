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

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var upstreamManifestCache = cache.NewCache[stremio.Manifest](&cache.CacheConfig{
	Name:          "stremio:wrap:upstreamManifest",
	Lifetime:      6 * time.Hour,
	LocalCapacity: 1024,
})

var upstreamResolverCache = cache.NewCache[upstreamsResolver](&cache.CacheConfig{
	Name:          "stremio:wrap:upstreamResolver",
	Lifetime:      24 * time.Hour,
	LocalCapacity: 2048,
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
	URL              string                                  `json:"u"`
	baseUrl          *url.URL                                `json:"-"`
	ExtractorId      string                                  `json:"e,omitempty"`
	extractor        stremio_transformer.StreamExtractorBlob `json:"-"`
	NoContentProxy   bool                                    `json:"ncp,omitempty"`
	ReconfigureStore bool                                    `json:"rs,omitempty"`
}

type UserData struct {
	Upstreams   []UserDataUpstream `json:"upstreams"`
	ManifestURL string             `json:"manifest_url,omitempty"`

	IncludeTorz bool `json:"torz,omitempty"`

	stremio_userdata.UserDataStores
	StoreName  string `json:"store,omitempty"`
	StoreToken string `json:"token,omitempty"`

	CachedOnly bool `json:"cached,omitempty"`

	TemplateId string                                 `json:"template,omitempty"`
	template   stremio_transformer.StreamTemplateBlob `json:"-"`

	Sort   string `json:"sort,omitempty"`
	Filter string `json:"filter,omitempty"`

	RPDBAPIKey       string `json:"rpdb_akey,omitempty"`
	TopPostersAPIKey string `json:"top_posters_akey,omitempty"`

	encoded   string             `json:"-"` // correctly configured
	manifests []stremio.Manifest `json:"-"`
	resolver  upstreamsResolver  `json:"-"`
}

func (ud UserData) StripSecrets() UserData {
	ud.UserDataStores = ud.UserDataStores.StripSecrets()
	ud.StoreToken = ""
	ud.RPDBAPIKey = ""
	ud.TopPostersAPIKey = ""
	return ud
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

func verifyTopPostersAPIKey(apiKey string) error {
	resp, err := config.DefaultHTTPClient.Get("https://api.top-streaming.stream/auth/verify/" + apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("Invalid API key")
	}
	return nil
}

type userDataError struct {
	upstreamUrl      []string
	store            []string
	token            []string
	rpdb_akey        string
	top_posters_akey string
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

	udErr := &userDataError{}
	if ud.RPDBAPIKey != "" && ud.TopPostersAPIKey != "" {
		err := "Only one poster provider can be used at a time"
		udErr.rpdb_akey = err
		udErr.top_posters_akey = err
		return ctx, udErr
	}

	if ud.TopPostersAPIKey != "" && udErr.top_posters_akey == "" {
		if err := verifyTopPostersAPIKey(ud.TopPostersAPIKey); err != nil {
			udErr.top_posters_akey = "Failed to Verify: " + err.Error()
			return ctx, udErr
		}
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
		udErr.upstreamUrl = upstreamUrlErrors
		return ctx, udErr
	}

	if err, errField := ud.UserDataStores.Prepare(ctx); err != nil {
		switch errField {
		case "store":
			udErr.store = []string{err.Error()}
		case "token":
			udErr.token = []string{err.Error()}
		default:
			udErr.store = []string{err.Error()}
		}
		return ctx, udErr
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
}

func (ud UserData) getUpstreamManifests(ctx *context.StoreContext, useCache bool) ([]stremio.Manifest, []error) {
	if ud.manifests == nil {
		var wg sync.WaitGroup

		manifests := make([]stremio.Manifest, len(ud.Upstreams))
		errs := make([]error, len(ud.Upstreams))
		hasError := false
		for i := range ud.Upstreams {
			up := &ud.Upstreams[i]
			cacheKey := up.baseUrl.String()
			hasCached := useCache && upstreamManifestCache.Get(cacheKey, &manifests[i])
			if !hasCached {
				wg.Go(func() {
					res, err := addon.GetManifest(&stremio_addon.GetManifestParams{BaseURL: up.baseUrl, ClientIP: ctx.ClientIP})
					manifests[i] = res.Data
					errs[i] = err
					if err != nil {
						hasError = true
						if err := upstreamManifestCache.AddWithLifetime(cacheKey, manifests[i], 15*time.Minute); err != nil {
							log.Warn("failed to cache upstream manifest (empty)", "error", err, "host", up.baseUrl.Host)
						}
					} else {
						if err := upstreamManifestCache.Add(cacheKey, manifests[i]); err != nil {
							log.Warn("failed to cache upstream manifest", "error", err, "host", up.baseUrl.Host)
						}
					}
				})
			} else if manifests[i].ID == "" {
				hasError = true
				errs[i] = errors.New("failed to fetch manifest: " + up.baseUrl.Host)
			}
		}
		wg.Wait()

		ud.manifests = manifests

		if hasError {
			return manifests, errs
		}
	}

	return ud.manifests, nil
}

func (ud UserData) getUpstreamsResolver(ctx *context.StoreContext) (upstreamsResolver, error) {
	eud := ud.GetEncoded()

	if ud.resolver == nil {
		if upstreamResolverCache.Get(eud, &ud.resolver) {
			return ud.resolver, nil
		}

		manifests, errs := ud.getUpstreamManifests(ctx, true)

		manifestCount := 0
		resolver := upstreamsResolver{}
		entryIdxMap := map[string]int{}
		for mIdx := range manifests {
			m := &manifests[mIdx]
			if m.ID == "" {
				continue
			}
			manifestCount++
			for _, r := range m.Resources {
				if r.Name == stremio.ResourceNameAddonCatalog {
					continue
				}

				if r.Name == stremio.ResourceNameCatalog {
					for _, c := range m.Catalogs {
						if c.Id == catalog_id_calendar_videos || c.Id == catalog_id_last_videos {
							key := string(r.Name) + ":" + c.Type
							if _, found := resolver[key]; !found {
								resolver[key] = []upstreamsResolverEntry{}
							}
							idPrefixKey := key + ":" + c.Id
							if idx, found := entryIdxMap[idPrefixKey]; found {
								resolver[key][idx].Indices = append(resolver[key][idx].Indices, mIdx)
							} else {
								resolver[key] = append(resolver[key], upstreamsResolverEntry{
									Prefix:  c.Id,
									Indices: []int{mIdx},
								})
								entryIdxMap[idPrefixKey] = len(resolver[key]) - 1
							}
						}
					}
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

		if err := errors.Join(errs...); err != nil {
			if manifestCount == 0 {
				return nil, err
			}
			log.Error("failed to fetch some upstream manifests", "error", err)
			if err := upstreamResolverCache.AddWithLifetime(eud, resolver, 1*time.Hour); err != nil {
				return nil, err
			}
		} else {
			if err := upstreamResolverCache.Add(eud, resolver); err != nil {
				return nil, err
			}
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
		upstreamsCount := len(ud.Upstreams)
		if upstreamsCount == 1 {
			return ud.Upstreams, nil
		}

		resolver, err := ud.getUpstreamsResolver(ctx)
		if err != nil {
			return nil, err
		}
		return resolver.resolve(ud, rName, rType, id), nil
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

	log := server.GetReqCtx(r).Log

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if data.GetEncoded() == "" {
			return data, nil
		}

		if err := udManager.Resolve(data); err != nil {
			return nil, err
		}
		if data.encoded == "" {
			return data, nil
		}

		shouldResync := false
		if data.StoreToken != "" {
			data.Stores = []stremio_userdata.Store{
				{
					Code:  stremio_userdata.StoreCode(store.StoreName(data.StoreName).Code()),
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

		for i := range data.Upstreams {
			up := &data.Upstreams[i]

			if up.ExtractorId != "" {
				if config.IsPublicInstance {
					up.ExtractorId = getNewTransformerExtractorId(up.ExtractorId)
				}

				if extractor, err := getExtractor(up.ExtractorId); err != nil {
					LogError(r, fmt.Sprintf("failed to fetch extractor(%s)", up.ExtractorId), err)
				} else {
					up.extractor = extractor
				}
			}
		}

		if data.TemplateId != "" {
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

		data.IncludeTorz = r.Form.Get("torz") == "on"
		data.Sort = r.Form.Get("sort")
		data.Filter = r.Form.Get("filter")
		data.RPDBAPIKey = r.Form.Get("rpdb_akey")
		data.TopPostersAPIKey = r.Form.Get("top_posters_akey")

		data.TemplateId = r.Form.Get("transformer.template_id")
		data.template = stremio_transformer.StreamTemplateBlob{
			Name:        r.Form.Get("transformer.template.name"),
			Description: r.Form.Get("transformer.template.description"),
		}

		for idx := range upstreams_length {
			upURL, err := stremio_addon.NormalizeManifestURL(r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].url"))
			if err != nil {
				log.Error("failed to normalize manifest url", "error", err)
			}
			extractorId := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].transformer.extractor_id")
			up := UserDataUpstream{
				URL:         upURL,
				ExtractorId: extractorId,
			}
			extractor := r.Form.Get("upstreams[" + strconv.Itoa(idx) + "].transformer.extractor")
			if extractor != "" {
				up.extractor = stremio_transformer.StreamExtractorBlob(extractor)
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
				data.Stores = []stremio_userdata.Store{
					{
						Code:  stremio_userdata.StoreCode(code),
						Token: token,
					},
				}
				break
			} else {
				data.Stores = append(data.Stores, stremio_userdata.Store{
					Code:  stremio_userdata.StoreCode(code),
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
			if baseUrl, err := stremio_addon.ExtractBaseURL(up.URL); err == nil {
				up.baseUrl = baseUrl
			}
		}
	}

	return data, nil
}
