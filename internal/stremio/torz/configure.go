package stremio_torz

import (
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

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

	action := r.Header.Get("x-addon-configure-action")
	if config.Stremio.Locked && !td.IsAuthed && IsMethod(r, http.MethodPost) {
		if action != "authorize" {
			sendPage(w, r, td)
			return
		}
	}

	if action != "" {
		switch action {
		case "authorize":
			if !IsPublicInstance {
				user := r.Form.Get("user")
				pass := r.Form.Get("pass")
				if pass == "" || config.Auth.GetPassword(user) != pass || !config.Auth.IsAdmin(user) {
					td.AuthError = "Wrong Credential!"
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
		case "add-indexer":
			if td.IsAuthed || len(td.Indexers) < MaxPublicInstanceIndexerCount {
				td.Indexers = append(td.Indexers, newTemplateDataIndexer(len(td.Indexers), "", "", ""))
			}

		case "remove-indexer":
			end := max(len(td.Indexers)-1, 0)
			td.Indexers = slices.Clone(td.Indexers[0:end])

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
		case "set-userdata-key":
			if td.IsAuthed {
				key := r.Form.Get("userdata_key")
				if key == "" {
					stremio_shared.RedirectToConfigurePage(w, r, "torz", "", false)
					return
				} else {
					err := udManager.Load(key, ud)
					if err != nil {
						LogError(r, "failed to load userdata", err)
					} else {
						stremio_shared.RedirectToConfigurePage(w, r, "torz", ud.GetEncoded(), false)
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
					stremio_shared.RedirectToConfigurePage(w, r, "torz", ud.GetEncoded(), true)
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
					stremio_shared.RedirectToConfigurePage(w, r, "torz", ud.GetEncoded(), false)
					return
				}
			}
		case "delete-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				err := udManager.Delete(ud)
				if err != nil {
					LogError(r, "failed to delete userdata", err)
				} else {
					eud := ""
					if !config.Stremio.Locked {
						eud = ud.GetEncoded()
					}
					stremio_shared.RedirectToConfigurePage(w, r, "torz", eud, false)
					return
				}
			}
		}

		sendPage(w, r, td)
		return
	}

	if ud.encoded != "" {
		_, err := ud.GetRequestContext(r)
		if err != nil {
			if uderr, ok := err.(*userDataError); ok {
				for i, err := range uderr.storeCode {
					td.Stores[i].Error.Code = err
				}
				for i, err := range uderr.storeToken {
					td.Stores[i].Error.Token = err
				}
			} else {
				SendError(w, r, err)
				return
			}
		}

		if !td.HasStoreError() && !ud.IsP2P() {
			s := ud.GetUser()
			if s.HasErr {
				for i, err := range s.Err {
					if err == nil {
						continue
					}
					log.Warn("failed to access store", "error", err)
					var ts *StoreConfig
					if ud.IsStremThruStore() {
						ts = &td.Stores[0]
						if ts.Error.Token != "" {
							ts.Error.Token += "\n"
						}
						ts.Error.Token += string(ud.GetStoreByIdx(i).Store.GetName()) + ": Failed to access store (" + err.Error() + ")"
					} else {
						ts = &td.Stores[i]
						ts.Error.Token = "Failed to access store (" + err.Error() + ")"
					}
				}
			}
		}

		for i := range ud.Indexers {
			indexer := &ud.Indexers[i]
			field, err := indexer.Validate()
			switch field {
			case "name":
				td.Indexers[i].Name.Error = err.Error()
			case "url":
				td.Indexers[i].URL.Error = err.Error()
			case "apikey":
				td.Indexers[i].APIKey.Error = err.Error()
			}
		}
	}

	hasError := td.HasFieldError()

	if IsMethod(r, http.MethodPost) && !td.IsAuthed && td.SavedUserDataKey != "" {
		server.ErrorForbidden(r).Send(w, r)
		return
	}

	if IsMethod(r, http.MethodPost) && !hasError {
		err = udManager.Sync(ud)
		if err != nil {
			SendError(w, r, err)
			return
		}

		stremio_shared.RedirectToConfigurePage(w, r, "torz", ud.GetEncoded(), td.SavedUserDataKey == "")
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/torz/" + ud.GetEncoded() + "/manifest.json").String()
	}

	sendPage(w, r, td)
}
