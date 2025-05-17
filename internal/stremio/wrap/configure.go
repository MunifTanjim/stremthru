package stremio_wrap

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
)

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
					stremio_shared.SetAdminCookie(w, user, pass)
					td.IsAuthed = true
					if r.Header.Get("hx-request") == "true" {
						w.Header().Add("hx-refresh", "true")
					}
				}
			}
		case "deauthorize":
			stremio_shared.UnsetAdminCookie(w)
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
			if td.IsAuthed || len(td.Stores) < MaxPublicInstanceStoreCount {
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
				up.Extractor = stremio_transformer.StreamExtractorBlob("")
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
				td.Template = stremio_transformer.StreamTemplateBlob{}
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
						td.Template = stremio_transformer.StreamTemplateBlob{}
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
		case "copy-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				name := r.Form.Get("userdata_name")
				ud.SetEncoded("")
				err := udManager.Save(ud, name)
				if err != nil {
					LogError(r, "failed to copy userdata", err)
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
				s := ud.GetUser()
				if s.HasErr {
					for i, err := range s.Err {
						LogError(r, "failed to access store", err)
						if err == nil {
							continue
						}
						var ts *StoreConfig
						if ud.IsStremThruStore() {
							ts = &td.Stores[0]
							if ts.Error.Token != "" {
								ts.Error.Token += "\n"
							}
							ts.Error.Token += string(ud.GetStoreByIdx(i).Store.GetName()) + ": Failed to access store"
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
