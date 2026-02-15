package stremio_list

import (
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	isAuthed := false
	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		isAuthed = config.UserAuth.GetPassword(cookie.User()) == cookie.Pass()
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

	if action := stremio_shared.GetConfigureAction(r); action != "" {
		switch action {
		case "authorize":
			if !IsPublicInstance {
				user := r.Form.Get("user")
				pass := r.Form.Get("pass")
				if pass == "" || config.AdminPassword.GetPassword(user) != pass {
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
		case "add-list":
			if td.IsAuthed || len(td.Lists) < MaxPublicInstanceListCount {
				idx := util.SafeParseInt(r.Header.Get("x-addon-configure-action-data"), -1)
				if idx >= len(td.Lists) {
					td.Lists = append(td.Lists, newTemplateDataList(len(td.Lists)))
				} else if idx != -1 {
					td.Lists = slices.Insert(td.Lists, idx+1, newTemplateDataList(idx+1))
				}
			}
		case "remove-list":
			idx := util.SafeParseInt(r.Header.Get("x-addon-configure-action-data"), -1)
			if idx != -1 && idx < len(td.Lists) {
				td.Lists = slices.Delete(td.Lists, idx, idx+1)
			}
		case "move-list-up":
			id := r.Header.Get("x-addon-configure-action-data")
			idx := slices.IndexFunc(td.Lists, func(tdl TemplateDataList) bool {
				return tdl.Id == id
			})
			if idx > 0 && idx < len(td.Lists) {
				td.Lists[idx], td.Lists[idx-1] = td.Lists[idx-1], td.Lists[idx]
			}
		case "move-list-down":
			id := r.Header.Get("x-addon-configure-action-data")
			idx := slices.IndexFunc(td.Lists, func(tdl TemplateDataList) bool {
				return tdl.Id == id
			})
			if idx >= 0 && idx < len(td.Lists)-1 {
				td.Lists[idx], td.Lists[idx+1] = td.Lists[idx+1], td.Lists[idx]
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
						stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), false)
						return
					}
				} else {
					err := udManager.Load(key, ud)
					if err != nil {
						LogError(r, "failed to load userdata", err)
					} else {
						stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), false)
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
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), true)
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
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), true)
					return
				}
			}
		case "delete-userdata":
			if td.IsAuthed && udManager.IsSaved(ud) {
				err := udManager.Delete(ud)
				if err != nil {
					LogError(r, "failed to delete userdata", err)
				} else {
					stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), true)
					return
				}
			}
		case "export":
			if ud.HasRequiredValues() {
				conf, err := udManager.Export(ud)
				if err != nil {
					SendError(w, r, err)
					return
				}
				td.ShareableConfig = conf
			}
		case "import":
			ud := &UserData{}
			if err = udManager.Import(r.Header.Get("x-addon-configure-action-data"), ud); err != nil {
				SendError(w, r, err)
				return
			}
			td = getTemplateData(ud, userDataError{}, td.IsAuthed, r)
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	if ud.GetEncoded() != "" || IsMethod(r, http.MethodPost) {
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

		stremio_shared.RedirectToConfigurePage(w, r, "list", ud.GetEncoded(), true)
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
