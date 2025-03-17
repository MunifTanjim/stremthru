package stremio_wrap

import (
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"golang.org/x/sync/singleflight"
)

var IsPublicInstance = config.IsPublicInstance
var MaxPublicInstanceUpstreamCount = 3
var MaxPublicInstanceStoreCount = 3

var addon = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/wrap/configure", http.StatusFound)
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

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	manifests, errs := ud.getUpstreamManifests(ctx)
	if errs != nil {
		serr := shared.ErrorInternalServerError(r, "failed to fetch upstream manifests")
		serr.Cause = errors.Join(errs...)
		serr.Send(w, r)
		return
	}

	manifest := GetManifest(r, manifests, ud)

	SendResponse(w, r, 200, manifest)
}

func redirectToConfigurePage(w http.ResponseWriter, r *http.Request, ud *UserData, tryInstall bool) {
	url := ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + ud.GetEncoded() + "/configure")
	if tryInstall {
		w.Header().Add("hx-trigger", "try_install")
	}

	if r.Header.Get("hx-request") == "true" {
		w.Header().Add("hx-location", url.String())
		w.WriteHeader(200)
	} else {
		http.Redirect(w, r, url.String(), http.StatusFound)
	}
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

	td := getTemplateData(ud, w, r)
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "cached":
			if ud.CachedOnly {
				conf.Default = "checked"
			}
		}
	}

	if action := r.Header.Get("x-addon-configure-action"); action != "" {
		switch action {
		case "authorize":
			if !IsPublicInstance {
				user := r.Form.Get("user")
				pass := r.Form.Get("pass")
				if pass == "" || config.ProxyAuthPassword.GetPassword(user) != pass {
					td.AuthError = "Wrong Credential!"
				} else if !config.AuthAdmin.IsAdmin(user) {
					td.AuthError = "Not Authorized!"
				} else {
					setCookie(w, user, pass)
					td.IsAuthed = true
					if r.Header.Get("hx-request") == "true" {
						w.Header().Add("hx-refresh", "true")
					}
				}
			}
		case "deauthorize":
			unsetCookie(w)
			td.IsAuthed = false
		case "add-upstream":
			if td.IsAuthed || len(td.Upstreams) < MaxPublicInstanceUpstreamCount {
				td.Upstreams = append(td.Upstreams, UpstreamAddon{
					URL: "",
				})
			}
		case "remove-upstream":
			end := len(td.Upstreams) - 1
			if end == 0 {
				end = 1
			}
			td.Upstreams = slices.Clone(td.Upstreams[0:end])
		case "add-store":
			if td.IsAuthed || len(td.Upstreams) < MaxPublicInstanceStoreCount {
				td.Stores = append(td.Stores, StoreConfig{})
			}
		case "remove-store":
			end := len(td.Stores) - 1
			if end == 0 {
				end = 1
			}
			td.Stores = slices.Clone(td.Stores[0:end])
		case "set-extractor":
			idx, err := strconv.Atoi(r.Form.Get("upstream_index"))
			if err != nil {
				SendError(w, r, err)
				return
			}
			up := &td.Upstreams[idx]
			id := up.ExtractorId
			if id != "" {
				value, err := getExtractor(id)
				if err != nil {
					LogError(r, "failed to fetch extractor", err)
					up.ExtractorError = "Failed to fetch extractor"
				} else if _, err := value.Parse(); err != nil {
					LogError(r, "failed to parse extractor", err)
					up.ExtractorError = err.Error()
				}
				up.Extractor = value
			} else {
				up.Extractor = StreamTransformerExtractorBlob("")
			}
		case "save-extractor":
			if td.IsAuthed {
				id := r.Form.Get("extractor_id")
				idx, err := strconv.Atoi(r.Form.Get("extractor_upstream_index"))
				if err != nil {
					SendError(w, r, err)
					return
				}
				up := &td.Upstreams[idx]
				value := up.Extractor
				if strings.HasPrefix(id, BUILTIN_TRANSFORMER_ENTITY_ID_EMOJI) {
					up.ExtractorError = "✨-prefixed ids are reserved"
				}
				if up.ExtractorError == "" {
					if _, err := value.Parse(); err != nil {
						LogError(r, "failed to parse extractor", err)
						up.ExtractorError = err.Error()
					} else {
						up.ExtractorId = id
						up.Extractor = value
					}
				}
				if up.ExtractorError == "" {
					if value == "" {
						if err := extractorStore.Del(id); err != nil {
							LogError(r, "failed to delete extractor", err)
							up.ExtractorError = "Failed to delete extractor"
						}
						extractorIds := []string{}
						for _, extractorId := range td.ExtractorIds {
							if extractorId != id {
								extractorIds = append(extractorIds, extractorId)
							}
						}
						td.ExtractorIds = extractorIds
						for i := range td.Upstreams {
							up := &td.Upstreams[i]
							if up.ExtractorId == id {
								up.ExtractorId = ""
								up.Extractor = ""
							}
						}
					} else {
						if err := extractorStore.Set(id, value); err != nil {
							LogError(r, "failed to save extractor", err)
							up.ExtractorError = "Failed to save extractor"
						} else {
							extractorIds := []string{}
							for _, extractorId := range td.ExtractorIds {
								if extractorId != id {
									extractorIds = append(extractorIds, extractorId)
								}
							}
							extractorIds = append(extractorIds, id)
							td.ExtractorIds = extractorIds
						}
					}
				}
			}
		case "set-template":
			id := ud.TemplateId
			if id != "" {
				value, err := getTemplate(id)
				if err != nil {
					LogError(r, "failed to fetch template", err)
					td.TemplateError.Name = "Failed to fetch template"
					td.TemplateError.Description = "Failed to fetch template"
				} else if t, err := value.Parse(); err != nil {
					LogError(r, "failed to parse template", err)
					if t.Name == nil {
						td.TemplateError.Name = err.Error()
					} else {
						td.TemplateError.Description = err.Error()
					}
				}
				td.Template = value
			} else {
				td.Template = StreamTransformerTemplateBlob{}
			}
		case "save-template":
			if td.IsAuthed {
				id := r.Form.Get("template_id")
				value := td.Template
				if strings.HasPrefix(id, BUILTIN_TRANSFORMER_ENTITY_ID_EMOJI) {
					td.TemplateError.Name = "✨-prefixed ids are reserved"
					td.TemplateError.Description = "✨-prefixed ids are reserved"
				}
				if td.TemplateError.IsEmpty() {
					if t, err := value.Parse(); err != nil {
						LogError(r, "failed to parse template", err)
						if t.Name == nil {
							td.TemplateError.Name = err.Error()
						} else {
							td.TemplateError.Description = err.Error()
						}
					} else {
						td.TemplateId = id
					}
				}
				if td.TemplateError.IsEmpty() {
					if value.Name == "" && value.Description == "" {
						if err := templateStore.Del(id); err != nil {
							LogError(r, "failed to delete template", err)
							td.TemplateError.Name = "Failed to delete template"
							td.TemplateError.Description = "Failed to delete template"
						}
						templateIds := []string{}
						for _, templateId := range td.TemplateIds {
							if templateId != id {
								templateIds = append(templateIds, templateId)
							}
						}
						td.TemplateIds = templateIds
						td.TemplateId = ""
						td.Template = StreamTransformerTemplateBlob{}
					} else {
						if err := templateStore.Set(id, value); err != nil {
							LogError(r, "failed to save template", err)
							td.TemplateError.Name = "Failed to save template"
							td.TemplateError.Description = "Failed to save template"
						} else {
							templateIds := []string{}
							for _, templateId := range td.TemplateIds {
								if templateId != id {
									templateIds = append(templateIds, templateId)
								}
							}
							templateIds = append(templateIds, id)
							td.TemplateIds = templateIds
						}
					}
				}
			}
		case "set-userdata-key":
			if td.IsAuthed {
				key := r.Form.Get("userdata_key")
				if key == "" {
					ud.SetEncoded("")
					err := udManager.Sync(ud)
					if err != nil {
						LogError(r, "failed to unselect userdata", err)
					} else {
						redirectToConfigurePage(w, r, ud, false)
						return
					}
				} else {
					err := udManager.Load(key, ud)
					if err != nil {
						LogError(r, "failed to load userdata", err)
					} else {
						redirectToConfigurePage(w, r, ud, false)
						return
					}
				}
			}
		case "save-userdata":
			if td.IsAuthed && !udManager.IsSaved(ud) && ud.HasRequiredValues() {
				name := r.Form.Get("userdata_name")
				err := udManager.Save(ud, name)
				if err != nil {
					LogError(r, "failed to save userdata", err)
				} else {
					redirectToConfigurePage(w, r, ud, true)
					return
				}
			}
		case "delete-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				err := udManager.Delete(ud)
				if err != nil {
					LogError(r, "failed to delete userdata", err)
				} else {
					redirectToConfigurePage(w, r, ud, true)
					return
				}
			}
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	if IsMethod(r, http.MethodPost) && !td.IsAuthed && td.SavedUserDataKey != "" {
		shared.ErrorForbidden(r).Send(w, r)
		return
	}

	if ud.encoded != "" {
		ctx, err := ud.GetRequestContext(r)
		if err != nil {
			if uderr, ok := err.(*userDataError); ok {
				for i, err := range uderr.upstreamUrl {
					td.Upstreams[i].Error = err
				}
				for i, err := range uderr.store {
					td.Stores[i].Error.Code = err
				}
				for i, err := range uderr.token {
					td.Stores[i].Error.Token = err
				}
			} else {
				SendError(w, r, err)
				return
			}
		}

		if !td.HasUpstreamError() {
			manifests, errs := ud.getUpstreamManifests(ctx)
			for i := range manifests {
				tup := &td.Upstreams[i]
				manifest := manifests[i]

				if tup.Error == "" {
					if errs != nil && errs[i] != nil {
						LogError(r, "failed to fetch manifest", errs[i])
						tup.Error = "Failed to fetch Manifest"
						continue
					}

					if manifest.BehaviorHints != nil && manifest.BehaviorHints.Configurable {
						tup.IsConfigurable = true
						if manifest.BehaviorHints.ConfigurationRequired {
							tup.Error = "Configuration Required"
							continue
						}
					}
				}
			}

			if !td.HasStoreError() {
				s := ud.stores.GetUser()
				if s.HasErr {
					for i, err := range s.Err {
						LogError(r, "failed to access store", err)
						if err == nil {
							continue
						}
						var ts *StoreConfig
						if ud.isStremThruStore {
							ts = &td.Stores[0]
							if ts.Error.Token != "" {
								ts.Error.Token += "\n"
							}
							ts.Error.Token += string(ud.stores[i].store.GetName()) + ": Failed to access store"
						} else {
							ts = &td.Stores[i]
							ts.Error.Token = "Failed to access store"
						}
					}
				}
			}
		}
	}

	hasError := td.HasFieldError()

	if IsMethod(r, http.MethodPost) && !hasError {
		err = udManager.Sync(ud)
		if err != nil {
			SendError(w, r, err)
			return
		}

		redirectToConfigurePage(w, r, ud, td.SavedUserDataKey == "")
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + ud.GetEncoded() + "/manifest.json").String()
	}

	page, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, page)
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	resource := r.PathValue("resource")
	contentType := r.PathValue("contentType")
	id := r.PathValue("id")
	extra := r.PathValue("extra")

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		SendError(w, r, err)
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
			SendError(w, r, err)
		}
		return
	case stremio.ResourceNameStream:
		res, err := ud.fetchStream(ctx, r, contentType, id)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendResponse(w, r, 200, res)
		return

	case stremio.ResourceNameSubtitles:
		res, err := ud.fetchSubtitles(ctx, contentType, id, extra)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendResponse(w, r, 200, res)
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

func waitForMagnetStatus(ctx *context.StoreContext, m *store.GetMagnetData, status store.MagnetStatus, maxRetry int, retryInterval time.Duration) (*store.GetMagnetData, error) {
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
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	magnetHash := r.PathValue("magnetHash")
	fileName := r.PathValue("fileName")
	fileIdx := -1
	if idx, err := strconv.Atoi(r.PathValue("fileIdx")); err == nil {
		fileIdx = idx
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		LogError(r, "failed to get request context", err)
		shared.ErrorBadRequest(r, "failed to get request context").Send(w, r)
		return
	}

	query := r.URL.Query()

	ctx.Store, ctx.StoreAuthToken = ud.stores[0].store, ud.stores[0].authToken
	if len(ud.stores) > 1 {
		storeCode := store.StoreCode(strings.ToLower(query.Get("s")))
		for i := range ud.stores {
			us := &ud.stores[i]
			if us.store.GetName().Code() == storeCode {
				ctx.Store, ctx.StoreAuthToken = us.store, us.authToken
				break
			}
		}
	}

	cacheKey := strings.Join([]string{ctx.ClientIP, string(ctx.Store.GetName()), ctx.StoreAuthToken, magnetHash, strconv.Itoa(fileIdx), fileName, query.Encode()}, ":")

	stremLink := ""
	if stremLinkCache.Get(cacheKey, &stremLink) {
		log.Debug("redirecting to cached stream link")
		http.Redirect(w, r, stremLink, http.StatusFound)
		return
	}

	result, err, _ := stremGroup.Do(cacheKey, func() (any, error) {
		log.Debug("creating stream link")
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

		sid := query.Get("sid")
		if sid == "" {
			sid = "*"
		}
		var pattern *regexp.Regexp
		if re := query.Get("re"); re != "" {
			if pat, err := regexp.Compile(re); err == nil {
				pattern = pat
			}
		}

		go buddy.TrackMagnet(ctx.Store, magnet.Hash, magnet.Files, sid, magnet.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)

		var file *store.MagnetFile
		if fileName != "" {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if f.Name == fileName {
					file = f
					log.Debug("matched file using filename", "filename", f.Name)
					break
				}
			}
		}
		if file == nil && pattern != nil {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if pattern.MatchString(f.Name) {
					file = f
					log.Debug("matched file using pattern", "pattern", pattern.String(), "filename", f.Name)
					break
				}
			}
		}
		if file == nil && strings.Contains(sid, ":") {
			if parts := strings.SplitN(sid, ":", 3); len(parts) == 3 {
				if pat, err := regexp.Compile("0?" + parts[1] + ".{1,3}" + "0?" + parts[2]); err == nil {
					for i := range magnet.Files {
						f := &magnet.Files[i]
						if pat.MatchString(f.Name) {
							file = f
							log.Debug("matched file using stream id", "sid", sid, "pattern", pat.String(), "filename", f.Name)
							break
						}
					}
				}
			}
		}
		if file == nil && fileIdx != -1 {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if f.Idx == fileIdx {
					file = f
					log.Debug("matched file using fileidx", "fileidx", f.Idx, "filename", f.Name)
					break
				}
			}
		}
		if file == nil {
			for i := range magnet.Files {
				f := &magnet.Files[i]
				if file == nil || file.Size < f.Size {
					file = f
					log.Debug("matched file using largest size", "filename", f.Name)
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
			LogError(r, strem.error_log, err)
		} else {
			log.Error(strem.error_log)
		}
		redirectToStaticVideo(w, r, cacheKey, strem.error_video)
		return
	}

	log.Debug("redirecting to stream link")
	http.Redirect(w, r, strem.link, http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioWrapEndpoints(mux *http.ServeMux) {
	seedDefaultTransformerEntities()

	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/{resource}/{contentType}/{id}", withCors(handleResource))
	router.HandleFunc("/{userData}/{resource}/{contentType}/{id}/{extra}", withCors(handleResource))

	router.HandleFunc("/{userData}/_/strem/{magnetHash}/{fileIdx}/{$}", withCors(handleStrem))
	router.HandleFunc("/{userData}/_/strem/{magnetHash}/{fileIdx}/{fileName}", withCors(handleStrem))

	mux.Handle("/stremio/wrap/", http.StripPrefix("/stremio/wrap", commonMiddleware(router)))
}
