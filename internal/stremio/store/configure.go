package stremio_store

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/store"
)

func validateUserdata(r *http.Request, ud *UserData, td *TemplateData) error {
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

	idr := ParsedId{isST: ud.StoreName == ""}
	if idr.isST && ud.EnableUsenet {
		idr.storeName = store.StoreNameStremThru
	} else {
		idr.storeName = store.StoreName(ud.StoreName)
	}
	idr.storeCode = idr.storeName.Code()
	ctx, err := ud.GetRequestContext(r, &idr)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			if uderr.storeName != "" {
				name_config.Error = uderr.storeName
			}
			if uderr.storeToken != "" {
				token_config.Error = uderr.storeToken
			}
		} else {
			return err
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

	return nil
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
			if !config.IsPublicInstance {
				user := r.FormValue("user")
				pass := r.FormValue("pass")
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
		case "set-userdata-key":
			if td.IsAuthed {
				key := r.FormValue("userdata_key")
				if key == "" {
					stremio_shared.RedirectToConfigurePage(w, r, "store", "", false)
					return
				} else {
					err := udManager.Load(key, ud)
					if err != nil {
						LogError(r, "failed to load userdata", err)
					} else {
						stremio_shared.RedirectToConfigurePage(w, r, "store", ud.GetEncoded(), false)
						return
					}
				}
			}
		case "save-userdata":
			if td.IsAuthed && !udManager.IsSaved(ud) {
				if err := validateUserdata(r, ud, td); err != nil {
					SendError(w, r, err)
					return
				}
				if !td.HasError() {
					name := r.FormValue("userdata_name")
					err := udManager.Save(ud, name)
					if err != nil {
						LogError(r, "failed to save userdata", err)
					} else {
						stremio_shared.RedirectToConfigurePage(w, r, "store", ud.GetEncoded(), true)
						return
					}
				}
			}
		case "copy-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				name := r.FormValue("userdata_name")
				ud.SetEncoded("")
				err := udManager.Save(ud, name)
				if err != nil {
					LogError(r, "failed to copy userdata", err)
				} else {
					stremio_shared.RedirectToConfigurePage(w, r, "store", ud.GetEncoded(), false)
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
					stremio_shared.RedirectToConfigurePage(w, r, "store", eud, false)
					return
				}
			}
		}

		sendPage(w, r, td)
		return
	}

	if ud.encoded != "" {
		if err := validateUserdata(r, ud, td); err != nil {
			SendError(w, r, err)
			return
		}
	}

	hasError := td.HasError()

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

		stremio_shared.RedirectToConfigurePage(w, r, "store", ud.GetEncoded(), td.SavedUserDataKey == "")
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + ud.GetEncoded() + "/manifest.json").String()
	}

	sendPage(w, r, td)
}
