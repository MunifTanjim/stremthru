package stremio_list

import (
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	isAuthed := false
	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		isAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	ud, err := getUserData(r, isAuthed)
	udErr := userDataError{}
	if err != nil {
		if e, ok := err.(userDataError); !ok {
			SendError(w, r, err)
			return
		} else {
			udErr = e
		}
	}

	td := getTemplateData(ud, udErr, isAuthed, r)

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
		case "add-list":
			if td.IsAuthed || len(td.Lists) < MaxPublicInstanceListCount {
				td.Lists = append(td.Lists, TemplateDataList{
					URL: "",
				})
			}
		case "remove-list":
			if end := len(td.Lists); end > 1 {
				td.Lists = slices.Clone(td.Lists[0 : end-1])
			}
		case "import-mdblist-mylists":
			if ud.MDBListAPIkey != "" {
				params := &mdblist.GetMyListsParams{}
				params.APIKey = ud.MDBListAPIkey
				res, err := mdblistClient.GetMyLists(params)
				if err != nil {
					SendError(w, r, err)
					return
				}
				lists := res.Data
				for i := range lists {
					list := lists[i]
					url := list.GetURL()
					if !slices.ContainsFunc(td.Lists, func(list TemplateDataList) bool {
						return list.URL == url
					}) {
						td.Lists = append(td.Lists, TemplateDataList{
							URL: url,
						})
					}
				}
				if !td.IsAuthed && len(lists) > MaxPublicInstanceListCount {
					td.Lists = td.Lists[0:MaxPublicInstanceListCount]
				}
			} else if td.MDBListAPIKey.Error == "" {
				td.MDBListAPIKey.Error = "Missing API Key"
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
						stremio_shared.RedirectToConfigurePage(w, r, "list", ud, false)
						return
					}
				} else {
					err := udManager.Load(key, ud)
					if err != nil {
						LogError(r, "failed to load userdata", err)
					} else {
						stremio_shared.RedirectToConfigurePage(w, r, "list", ud, false)
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
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud, true)
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
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud, true)
					return
				}
			}
		case "delete-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				err := udManager.Delete(ud)
				if err != nil {
					LogError(r, "failed to delete userdata", err)
				} else {
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud, true)
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

	if ud.GetEncoded() != "" {
		if len(td.Lists) == 0 {
			list := TemplateDataList{}
			list.Error.URL = "Missing List URL"
			td.Lists = append(td.Lists, list)
		}
	}

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/list/" + ud.GetEncoded() + "/manifest.json").String()
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	hasError := td.HasFieldError()

	if IsMethod(r, http.MethodPost) && !hasError {
		err = udManager.Sync(ud)
		if err != nil {
			SendError(w, r, err)
			return
		}

		stremio_shared.RedirectToConfigurePage(w, r, "list", ud, true)
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/list/" + ud.GetEncoded() + "/manifest.json").String()
	}

	page, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, page)
}
