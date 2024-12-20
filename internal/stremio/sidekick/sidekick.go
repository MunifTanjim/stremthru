package stremio_sidekick

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/stremio/api"
)

var client = func() *stremio_api.Client {
	return stremio_api.NewClient(&stremio_api.ClientConfig{})
}()

var addon_client = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

type CookieValue struct {
	url.Values
	IsExpired bool
}

func (cv *CookieValue) AuthKey() string {
	return cv.Get("auth_key")
}

func (cv *CookieValue) Email() string {
	return cv.Get("email")
}

const COOKIE_NAME = "stremio.sidekick.auth"
const COOKIE_PATH = "/stremio/sidekick/"

func setCookie(w http.ResponseWriter, authKey string, email string) {
	value := &url.Values{
		"auth_key": []string{authKey},
		"email":    []string{email},
	}
	cookie := &http.Cookie{
		Name:     COOKIE_NAME,
		Value:    value.Encode(),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     COOKIE_PATH,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

func unsetCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Expires: time.Unix(0, 0),
		Path:    COOKIE_PATH,
	})
}

func getCookieValue(w http.ResponseWriter, r *http.Request) (*CookieValue, error) {
	cookie, err := r.Cookie(COOKIE_NAME)
	value := &CookieValue{}
	if err != nil {
		if err != http.ErrNoCookie {
			return value, err
		}
		value.IsExpired = true
		return value, nil
	}

	v, err := url.ParseQuery(cookie.Value)
	if err != nil {
		unsetCookie(w)
		value.IsExpired = true
		return value, nil
	}
	value.Values = v
	return value, nil
}

func getTemplateData(cookie *CookieValue, r *http.Request) *TemplateData {
	td := &TemplateData{
		Title: "Stremio Sidekick",
	}
	if cookie != nil && !cookie.IsExpired {
		td.IsAuthed = true
		td.Email = cookie.Email()
	}

	td.AddonOperation = r.URL.Query().Get("addon_operation")
	if td.AddonOperation == "" {
		hxCurrUrl := r.Header.Get("hx-current-url")
		if hxCurrUrl != "" {
			if hxUrl, err := url.Parse(hxCurrUrl); err == nil {
				td.AddonOperation = hxUrl.Query().Get("addon_operation")
			}
		}
	}
	return td
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, err)
		return
	}

	td := getTemplateData(cookie, r)

	buf, err := GetPage(td)
	if err != nil {
		SendError(w, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	res, err := client.Login(&stremio_api.LoginParams{
		Email:    email,
		Password: password,
	})
	if err != nil {
		SendError(w, err)
		return
	}

	setCookie(w, res.Data.AuthKey, res.Data.User.Email)

	http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	unsetCookie(w)

	http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
}

func handleAddons(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, err)
		return
	}

	td := getTemplateData(cookie, r)
	td.Addons = res.Data.Addons

	buf, err := ExecuteTemplate(td, "addons_section.html")
	if err != nil {
		SendError(w, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonMove(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	transportUrl := r.PathValue("transportUrl")
	direction := r.PathValue("direction")

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, err)
		return
	}

	currAddons := get_res.Data.Addons
	totalAddons := len(currAddons)

	td := getTemplateData(cookie, r)
	td.Addons = make([]stremio_api.Addon, 0, totalAddons)
	td.Addons = append(td.Addons, currAddons...)

	idx := -1
	for i := range td.Addons {
		if td.Addons[i].TransportUrl == transportUrl {
			idx = i
			break
		}
	}

	if idx != -1 {
		switch direction {
		case "top":
			if idx == 0 {
				break
			}
			addons := make([]stremio_api.Addon, 0, totalAddons)
			addons = append(addons, td.Addons[idx])
			addons = append(addons, td.Addons[:idx]...)
			addons = append(addons, td.Addons[idx+1:]...)
			td.Addons = addons
		case "up":
			if idx == 0 {
				break
			}
			td.Addons[idx], td.Addons[idx-1] = td.Addons[idx-1], td.Addons[idx]
		case "down":
			if idx == totalAddons-1 {
				break
			}
			td.Addons[idx], td.Addons[idx+1] = td.Addons[idx+1], td.Addons[idx]
		case "bottom":
			if idx == totalAddons-1 {
				break
			}
			addons := make([]stremio_api.Addon, 0, totalAddons)
			addons = append(addons, td.Addons[:idx]...)
			addons = append(addons, td.Addons[idx+1:]...)
			addons = append(addons, td.Addons[idx])
			td.Addons = addons
		}

		set_params := &stremio_api.SetAddonsParams{
			Addons: td.Addons,
		}
		set_params.APIKey = cookie.AuthKey()
		set_res, err := client.SetAddons(set_params)
		if err != nil || !set_res.Data.Success {
			td.Addons = currAddons
		}
	}

	buf, err := ExecuteTemplate(td, "addons_section.html")
	if err != nil {
		SendError(w, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonToggle(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	transportUrl := r.PathValue("transportUrl")

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, err)
		return
	}

	currAddons := get_res.Data.Addons
	totalAddons := len(currAddons)

	td := getTemplateData(cookie, r)
	td.Addons = make([]stremio_api.Addon, 0, totalAddons)
	td.Addons = append(td.Addons, currAddons...)

	idx := -1
	for i := range td.Addons {
		if td.Addons[i].TransportUrl == transportUrl {
			idx = i
			break
		}
	}

	if idx != -1 {
		addon := &td.Addons[idx]
		isDisabled := strings.HasPrefix(addon.Manifest.ID, "st:disabled:")
		if isDisabled {
			if transportUrl, err := url.Parse(addon.TransportUrl); err == nil {
				transportUrl.Path = strings.TrimSuffix(transportUrl.Path, "/manifest.json")
				transportUrl.Path = strings.TrimPrefix(transportUrl.Path, "/stremio/disabled/")
				if transportUrl, err = url.Parse(transportUrl.Path); err == nil {
					transportUrl.Path = strings.TrimSuffix(transportUrl.Path, "/manifest.json")

					if manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{
						BaseURL: transportUrl,
					}); err == nil {
						enabledAddon := stremio_api.Addon{
							TransportUrl: transportUrl.JoinPath("manifest.json").String(),
							Manifest:     manifest.Data,
							Flags:        addon.Flags,
						}

						td.Addons[idx] = enabledAddon
					}
				}
			}
		} else {
			transportUrl := shared.ExtractRequestBaseURL(r)
			transportUrl.Path = "/stremio/disabled/" + url.PathEscape(addon.TransportUrl)
			transportUrl.RawPath = transportUrl.Path
			transportUrl.Path, _ = url.PathUnescape(transportUrl.Path)

			if manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{
				BaseURL: transportUrl,
			}); err == nil {
				disabledAddon := stremio_api.Addon{
					TransportUrl: transportUrl.JoinPath("manifest.json").String(),
					Manifest:     manifest.Data,
					Flags:        addon.Flags,
				}

				if !addon.Flags.Protected {
					td.Addons[idx] = disabledAddon
				}
			}
		}

		set_params := &stremio_api.SetAddonsParams{
			Addons: td.Addons,
		}
		set_params.APIKey = cookie.AuthKey()
		set_res, err := client.SetAddons(set_params)
		if err != nil || !set_res.Data.Success {
			td.Addons = currAddons
		}
	}

	buf, err := ExecuteTemplate(td, "addons_section.html")
	if err != nil {
		SendError(w, err)
		return
	}
	SendHTML(w, 200, buf)
}

func AddStremioSidekickEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/stremio/sidekick", handleRoot)
	mux.HandleFunc("/stremio/sidekick/{$}", handleRoot)

	mux.HandleFunc("/stremio/sidekick/login", handleLogin)
	mux.HandleFunc("/stremio/sidekick/logout", handleLogout)

	mux.HandleFunc("/stremio/sidekick/addons", handleAddons)
	mux.HandleFunc("/stremio/sidekick/addons/{transportUrl}/move/{direction}", handleAddonMove)
	mux.HandleFunc("/stremio/sidekick/addons/{transportUrl}/toggle", handleAddonToggle)
}
