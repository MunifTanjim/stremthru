package stremio_sidekick

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/stremio/api"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var IsPublicInstance = config.IsPublicInstance

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
		Expires:  time.Now().Add(7 * 24 * time.Hour),
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
		LogError(r, "failed to parse cookie value", err)
		unsetCookie(w)
		value.IsExpired = true
		return value, nil
	}
	value.Values = v
	return value, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(cookie, w, r)

	if action := r.Header.Get("x-addon-configure-action"); action != "" {
		switch action {
		case "authorize":
			if !IsPublicInstance {
				user := r.FormValue("user")
				pass := r.FormValue("pass")
				if pass == "" || config.ProxyAuthPassword.GetPassword(user) != pass {
					td.AuthAdminError = "Wrong Credential!"
				} else if !config.AuthAdmin.IsAdmin(user) {
					td.AuthAdminError = "Not Authorized!"
				} else {
					stremio_shared.SetAdminCookie(w, user, pass)
					td.HasAuthAdmin = true
					if r.Header.Get("hx-request") == "true" {
						w.Header().Add("hx-refresh", "true")
					}
				}
			}
		case "deauthorize":
			stremio_shared.UnsetAdminCookie(w)
			td.HasAuthAdmin = false
		}
	}

	buf, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	manifest := GetManifest(r)
	SendResponse(w, r, 200, manifest)
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	method := r.FormValue("method")

	if method == "password" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		res, err := client.Login(&stremio_api.LoginParams{
			Email:    email,
			Password: password,
		})
		if err == nil {
			setCookie(w, res.Data.AuthKey, res.Data.User.Email)
			if r.Header.Get("hx-request") == "true" {
				w.Header().Add("hx-refresh", "true")
				w.Header().Add("hx-redirect", "/stremio/sidekick")
				w.WriteHeader(200)
			} else {
				http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
			}
			return
		}

		if rerr, ok := err.(*stremio_api.ResponseError); ok {
			td := getTemplateData(nil, w, r)
			td.Login.Email = email
			td.Login.Password = password
			switch rerr.Code {
			case stremio_api.ErrorCodeUserNotFound:
				td.Login.Error.Email = rerr.Message
			case stremio_api.ErrorCodeWrongPassphrase:
				td.Login.Error.Password = rerr.Message
			}
			buf, err := executeTemplate(td, "sidekick_account_section.html")
			if err != nil {
				SendError(w, r, err)
				return
			}
			SendHTML(w, 200, buf)
		} else {
			SendError(w, r, err)
		}
	} else if method == "token" {
		token := r.FormValue("token")

		params := &stremio_api.GetUserParams{}
		params.APIKey = token
		res, err := client.GetUser(params)
		if err == nil {
			setCookie(w, token, res.Data.Email)
			if r.Header.Get("hx-request") == "true" {
				w.Header().Add("hx-refresh", "true")
				w.Header().Add("hx-redirect", "/stremio/sidekick")
				w.WriteHeader(200)
			} else {
				http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
			}
			return
		}

		if rerr, ok := err.(*stremio_api.ResponseError); ok {
			td := getTemplateData(nil, w, r)
			td.Login.Token = token
			switch rerr.Code {
			case stremio_api.ErrorCodeSessionNotFound:
				td.Login.Error.Token = rerr.Message
			}
			buf, err := executeTemplate(td, "sidekick_account_section.html")
			if err != nil {
				SendError(w, r, err)
				return
			}
			SendHTML(w, 200, buf)
		} else {
			SendError(w, r, err)
		}
	} else {
		shared.ErrorBadRequest(r, "invalid login method").Send(w, r)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	unsetCookie(w)

	http.Redirect(w, r, "/stremio/sidekick", http.StatusFound)
}

func handleAddons(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(cookie, w, r)
	td.Addons = res.Data.Addons

	if IsMethod(r, http.MethodPost) {
		manifestUrl, err := stremio_addon.NormalizeManifestURL(r.FormValue("manifest_url"))
		if err != nil {
			td.AddonError = "invalid manifest url"
			log.Error("failed to normalize manifest url", "error", err)
		} else if manifestUrl == "" {
			td.AddonError = "missing manifest url"
		}

		baseUrl, err := stremio_addon.ExtractBaseURL(manifestUrl)
		if err != nil && td.AddonError == "" {
			td.AddonError = "invalid manifest url"
			log.Error("failed to extract base url", "error", err)
		}

		if td.AddonError == "" {
			gmParams := &stremio_addon.GetManifestParams{BaseURL: baseUrl}
			gmParams.APIKey = cookie.AuthKey()
			gmRes, err := addon_client.GetManifest(gmParams)
			if err != nil {
				td.AddonError = fmt.Sprintf("failed to get manifest: %v", err)
				log.Error("failed to get manifest", "error", err)
			} else if slices.ContainsFunc(td.Addons, func(addon stremio.Addon) bool {
				return addon.TransportUrl == manifestUrl
			}) {
				td.AddonError = "addon already exists"
			} else {
				td.Addons = append(td.Addons, stremio.Addon{
					TransportUrl: manifestUrl,
					Manifest:     gmRes.Data,
					Flags:        &stremio.AddonFlags{},
				})

				saParams := &stremio_api.SetAddonsParams{Addons: td.Addons}
				saParams.APIKey = cookie.AuthKey()
				_, err := client.SetAddons(saParams)
				if err != nil {
					td.AddonError = fmt.Sprintf("failed to install addon: %v", err)
					log.Error("failed to install addon", "error", err)
					td.Addons = td.Addons[:len(td.Addons)-1]
				}
			}
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonsBackup(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	filename := "Stremio-Addons-" + cookie.Email() + "-" + strconv.FormatInt(res.Data.LastModified.UnixMilli(), 10) + ".json"
	w.Header().Add("Content-Disposition", `attachment; filename="`+filename+`"`)

	SendResponse(w, r, 200, res.Data)
}

func handleAddonsRestore(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(cookie, w, r)

	td.BackupRestore.AddonsRestoreBlob = r.FormValue("blob")

	backup := &stremio_api.GetAddonsData{}
	err = json.Unmarshal([]byte(td.BackupRestore.AddonsRestoreBlob), backup)
	if err != nil {
		td.BackupRestore.Error.AddonsRestoreBlob = "failed to parse: " + err.Error()
	}

	if td.BackupRestore.Error.AddonsRestoreBlob == "" {
		params := &stremio_api.SetAddonsParams{Addons: backup.Addons}
		params.APIKey = cookie.AuthKey()
		_, err := client.SetAddons(params)
		if err == nil {
			w.Header().Add("HX-Redirect", "/stremio/sidekick/?addon_operation=move&try_load_addons=1")
			SendResponse(w, r, 200, "")
			return
		}

		td.BackupRestore.Error.AddonsRestoreBlob = "failed to restore: " + err.Error()
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonUninstall(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodDelete) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	currAddons := get_res.Data.Addons

	td := getTemplateData(cookie, w, r)

	transportUrl := r.PathValue("transportUrl")

	td.Addons = slices.Clone(currAddons)
	td.Addons = slices.DeleteFunc(td.Addons, func(addon stremio.Addon) bool {
		return addon.TransportUrl == transportUrl
	})

	set_params := &stremio_api.SetAddonsParams{
		Addons: td.Addons,
	}
	set_params.APIKey = cookie.AuthKey()
	set_res, err := client.SetAddons(set_params)
	if err != nil {
		err = core.PackError(err)
		log.Error("failed to uninstall addons", "error", err)
		td.AddonError = fmt.Sprintf("failed to uninstall addons: %v", err)
		td.Addons = currAddons
	} else if !set_res.Data.Success {
		err_msg := "failed to uninstall addons!"
		log.Error(err_msg)
		td.AddonError = err_msg
		td.Addons = currAddons
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonsReset(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(cookie, w, r)
	understood := r.FormValue("understood") == "on"

	if !understood {
		td.BackupRestore.HasError.AddonsReset = true
		td.BackupRestore.Message.AddonsReset = "Missing Acknowledgement"
	} else {
		addons := []stremio.Addon{
			{
				Flags:         &stremio.AddonFlags{Official: true, Protected: true},
				TransportName: "",
				TransportUrl:  "https://v3-cinemeta.strem.io/manifest.json",
			},
			{
				Flags: &stremio.AddonFlags{Official: true, Protected: true},
				Manifest: stremio.Manifest{
					BehaviorHints: &stremio.BehaviorHints{},
					Catalogs:      []stremio.Catalog{},
					Description:   "Local add-on to find playable files: .torrent, .mp4, .mkv and .avi",
					ID:            "org.stremio.local",
					Name:          "Local Files (without catalog support)",
					Resources: []stremio.Resource{
						{
							IDPrefixes: []string{"local:", "bt:"},
							Name:       stremio.ResourceNameMeta,
							Types:      []stremio.ContentType{"other"},
						},
						{
							IDPrefixes: []string{"tt"},
							Name:       stremio.ResourceNameStream,
							Types:      []stremio.ContentType{stremio.ContentTypeMovie, stremio.ContentTypeSeries},
						},
					},
					Types:   []stremio.ContentType{stremio.ContentTypeMovie, stremio.ContentTypeSeries, "other"},
					Version: "1.10.0",
				},
				TransportName: "",
				TransportUrl:  "http://127.0.0.1:11470/local-addon/manifest.json",
			},
			{
				Flags: &stremio.AddonFlags{Official: true},
				Manifest: stremio.Manifest{
					BehaviorHints: &stremio.BehaviorHints{},
					Catalogs:      []stremio.Catalog{},
					Description:   "The official add-on for subtitles from OpenSubtitles",
					ID:            "org.stremio.opensubtitles",
					Logo:          "http://www.strem.io/images/addons/opensubtitles-logo.png",
					Name:          "OpenSubtitles",
					Resources: []stremio.Resource{
						{Name: stremio.ResourceNameSubtitles},
					},
					Types:   []stremio.ContentType{stremio.ContentTypeSeries, stremio.ContentTypeMovie, "other"},
					Version: "0.24.0",
				},
				TransportName: "",
				TransportUrl:  "https://opensubtitles.strem.io/stremio/v1",
			},
			{
				Flags:         &stremio.AddonFlags{Official: true},
				TransportName: "",
				TransportUrl:  "https://opensubtitles-v3.strem.io/manifest.json",
			},
			{
				Flags:         &stremio.AddonFlags{Official: true},
				TransportName: "",
				TransportUrl:  "https://caching.stremio.net/publicdomainmovies.now.sh/manifest.json",
			},
			{
				Flags:         &stremio.AddonFlags{Official: true},
				TransportName: "",
				TransportUrl:  "https://watchhub.strem.io/manifest.json",
			},
		}

		for i := range addons {
			addon := &addons[i]
			if addon.Manifest.ID != "" {
				continue
			}
			manifestUrl, err := url.Parse(addon.TransportUrl)
			if err != nil {
				log.Error("failed to parse manifest url", "error", err)
				td.BackupRestore.HasError.AddonsReset = true
				td.BackupRestore.Message.AddonsReset = "Failed to reset: " + err.Error()
				break
			}
			if manifestUrl.RawPath == "" {
				manifestUrl.RawPath = manifestUrl.Path
			}
			manifestUrl.RawPath = strings.TrimSuffix(manifestUrl.RawPath, "/manifest.json")
			manifestUrl.Path, _ = url.PathUnescape(manifestUrl.RawPath)
			manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{
				BaseURL: manifestUrl,
			})
			if err != nil {
				log.Error("failed to get manifest", "url", addon.TransportUrl, "error", err)
				td.BackupRestore.HasError.AddonsReset = true
				td.BackupRestore.Message.AddonsReset = "Failed to reset: " + err.Error()
				break
			}
			addon.Manifest = manifest.Data
		}

		if !td.BackupRestore.HasError.AddonsReset {
			params := &stremio_api.SetAddonsParams{Addons: addons}
			params.APIKey = cookie.AuthKey()
			_, err := client.SetAddons(params)
			if err == nil {
				w.Header().Add("HX-Redirect", "/stremio/sidekick/?addon_operation=move&try_load_addons=1")
				SendResponse(w, r, 200, "")
				return
			}

			log.Error("failed to set addons", "error", err)
			td.BackupRestore.HasError.AddonsReset = true
			td.BackupRestore.Message.AddonsReset = "Failed to reset: " + err.Error()
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonMove(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	transportUrl := r.PathValue("transportUrl")
	direction := r.PathValue("direction")

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	currAddons := get_res.Data.Addons
	totalAddons := len(currAddons)

	td := getTemplateData(cookie, w, r)
	td.Addons = make([]stremio.Addon, 0, totalAddons)
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
			addons := make([]stremio.Addon, 0, totalAddons)
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
			addons := make([]stremio.Addon, 0, totalAddons)
			addons = append(addons, td.Addons[:idx]...)
			addons = append(addons, td.Addons[idx+1:]...)
			addons = append(addons, td.Addons[idx])
			td.Addons = addons
		}

		if td.AddonError == "" {
			set_params := &stremio_api.SetAddonsParams{
				Addons: td.Addons,
			}
			set_params.APIKey = cookie.AuthKey()
			set_res, err := client.SetAddons(set_params)
			if err != nil {
				err = core.PackError(err)
				log.Error("failed to set addons", "error", err)
				td.AddonError = fmt.Sprintf("failed to set addons: %v", err)
				td.Addons = currAddons
			} else if !set_res.Data.Success {
				err_msg := fmt.Sprintf("failed to set addons!")
				log.Error(err_msg)
				td.AddonError = strings.TrimSpace(err_msg)
				td.Addons = currAddons
			}
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonReload(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	currAddons := get_res.Data.Addons
	totalAddons := len(currAddons)

	td := getTemplateData(cookie, w, r)
	td.Addons = make([]stremio.Addon, 0, totalAddons)
	td.Addons = append(td.Addons, currAddons...)

	transportUrl := r.PathValue("transportUrl")

	idx := -1
	for i := range td.Addons {
		if td.Addons[i].TransportUrl == transportUrl {
			idx = i
			break
		}
	}

	if idx != -1 {
		addon := &td.Addons[idx]

		manifestUrl := r.FormValue("manifest_url")
		if manifestUrl == "" {
			manifestUrl = addon.TransportUrl
		}
		manifestUrl, err = stremio_addon.NormalizeManifestURL(manifestUrl)
		if err != nil {
			err = core.PackError(err)
			log.Error("failed to normalize manifest url", "error", err)
			td.AddonError = fmt.Sprintf("failed to normalize manifest url: %v", err)
		}

		oldTransportUrl, err := url.Parse(addon.TransportUrl)
		if err != nil && td.AddonError == "" {
			err = core.PackError(err)
			log.Error("failed to parse old manifest url", "error", err)
			td.AddonError = fmt.Sprintf("failed to parse old manifest url: %v", err)
		}

		if transportUrl, err := url.Parse(manifestUrl); err == nil && td.AddonError == "" {
			rawPath := transportUrl.RawPath
			if rawPath == "" {
				rawPath = transportUrl.Path
			}
			transportUrl.RawPath = strings.TrimSuffix(rawPath, "/manifest.json")
			transportUrl.Path, _ = url.PathUnescape(transportUrl.RawPath)

			if transportUrl.Host != oldTransportUrl.Host {
				err_msg := fmt.Sprintf("manifest url host changed")
				log.Error(err_msg)
				td.AddonError = err_msg
			} else if manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{BaseURL: transportUrl}); err != nil {
				err = core.PackError(err)
				log.Error("failed to get manifest", "error", err)
				td.AddonError = fmt.Sprintf("failed to get manifest: %v", err)
			} else {
				refreshedAddon := stremio.Addon{
					TransportUrl:  manifestUrl,
					TransportName: addon.TransportName,
					Manifest:      manifest.Data,
					Flags:         addon.Flags,
				}

				if addon.Manifest.BehaviorHints != nil && !addon.Manifest.BehaviorHints.Configurable {
					if refreshedAddon.Manifest.BehaviorHints == nil {
						refreshedAddon.Manifest.BehaviorHints = &stremio.BehaviorHints{}
					}
					refreshedAddon.Manifest.BehaviorHints.Configurable = false
				}

				td.Addons[idx] = refreshedAddon
			}
		}

		set_params := &stremio_api.SetAddonsParams{
			Addons: td.Addons,
		}
		set_params.APIKey = cookie.AuthKey()
		set_res, err := client.SetAddons(set_params)
		if err != nil {
			err = core.PackError(err)
			log.Error("failed to set addons", "error", err)
			td.AddonError = fmt.Sprintf("failed to set addons: %v", err)
			td.Addons = currAddons
		} else if !set_res.Data.Success {
			err_msg := "failed to set addons!"
			log.Error(err_msg)
			td.AddonError = err_msg
			td.Addons = currAddons
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonToggle(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	transportUrl := r.PathValue("transportUrl")

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	currAddons := get_res.Data.Addons
	totalAddons := len(currAddons)

	td := getTemplateData(cookie, w, r)
	td.Addons = make([]stremio.Addon, 0, totalAddons)
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
					transportUrl = transportUrl.JoinPath("..")

					manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{
						BaseURL: transportUrl,
					})
					if err != nil {
						err = core.PackError(err)
						log.Error("failed to get manifest", "error", err)
						td.AddonError = fmt.Sprintf("failed to get manifest: %v", err)
					} else {
						enabledAddon := stremio.Addon{
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
			transportUrl.RawPath = "/stremio/disabled/" + url.PathEscape(addon.TransportUrl)
			transportUrl.Path = "/stremio/disabled/" + addon.TransportUrl

			manifest, err := addon_client.GetManifest(&stremio_addon.GetManifestParams{
				BaseURL: transportUrl,
			})
			if err != nil {
				err = core.PackError(err)
				log.Error("failed to get manifest", "error", err)
				td.AddonError = fmt.Sprintf("failed to get manifest: %v", err)
			} else {
				disabledAddon := stremio.Addon{
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
		if err != nil {
			err = core.PackError(err)
			log.Error("failed to set addons", "error", err)
			td.AddonError = fmt.Sprintf("failed to set addons: %v", err)
			td.Addons = currAddons
		} else if !set_res.Data.Success {
			err_msg := "failed to set addons!"
			log.Error(err_msg)
			td.AddonError = strings.TrimSpace(err_msg)
			td.Addons = currAddons
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleAddonModify(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	log := server.GetReqCtx(r).Log

	transportUrl := r.PathValue("transportUrl")
	name := r.FormValue("name")
	description := r.FormValue("description")
	logo := r.FormValue("logo")

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = cookie.AuthKey()
	get_res, err := client.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	currAddons := get_res.Data.Addons

	td := getTemplateData(cookie, w, r)
	td.Addons = slices.Clone(currAddons)

	idx := -1
	for i := range td.Addons {
		if td.Addons[i].TransportUrl == transportUrl {
			idx = i
			break
		}
	}

	if idx != -1 {
		addon := &td.Addons[idx]
		addon.Manifest.Name = name
		addon.Manifest.Description = description
		if logo != "" {
			addon.Manifest.Logo = logo
		}
		for i := range addon.Manifest.Catalogs {
			catalog := &addon.Manifest.Catalogs[i]
			if name := r.FormValue("catalog_name[" + strconv.Itoa(i) + "]"); name != "" {
				catalog.Name = name
			}
			if hidden := r.FormValue("catalog_hidden[" + strconv.Itoa(i) + "]"); name != "" {
				if canToggleCatalogBoard(catalog) {
					toggleCatalogBoard(catalog, hidden == "true")
				}
			}
		}
		if td.HasAuthAdmin {
			isConfigurable := r.FormValue("configurable") == "true"
			isProtected := r.FormValue("protected") == "true"

			if addon.Manifest.BehaviorHints != nil {
				addon.Manifest.BehaviorHints.Configurable = isConfigurable
			}
			if addon.Flags == nil {
				addon.Flags = &stremio.AddonFlags{}
			}
			if !addon.Flags.Official {
				addon.Flags.Protected = isProtected
			}
		}
		set_params := &stremio_api.SetAddonsParams{
			Addons: td.Addons,
		}
		set_params.APIKey = cookie.AuthKey()
		set_res, err := client.SetAddons(set_params)
		if err != nil {
			err = core.PackError(err)
			log.Error("failed to set addons", "error", err)
			td.AddonError = fmt.Sprintf("failed to set addons: %v", err)
			td.Addons = currAddons
		} else if !set_res.Data.Success {
			err_msg := "failed to set addons!"
			log.Error(err_msg)
			td.AddonError = err_msg
			td.Addons = currAddons
		}
	}

	buf, err := executeTemplate(td, "sidekick_addons_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleLibraryBackup(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAllLibraryItemsParams{}
	params.APIKey = cookie.AuthKey()
	res, err := client.GetAllLibraryItems(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	lastModified := time.Unix(0, 0)
	for i := range res.Data {
		item := res.Data[i]
		if item.MTime.After(lastModified) {
			lastModified = item.MTime
		}
	}

	filename := "Stremio-Library-" + cookie.Email() + "-" + strconv.FormatInt(lastModified.UnixMilli(), 10) + ".json"
	w.Header().Add("Content-Disposition", `attachment; filename="`+filename+`"`)

	SendResponse(w, r, 200, res.Data)
}

func handleLibraryRestore(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	cookie, err := getCookieValue(w, r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(cookie, w, r)

	td.BackupRestore.LibraryRestoreBlob = r.FormValue("blob")

	backup := &stremio_api.GetAllLibraryItemsData{}
	err = json.Unmarshal([]byte(td.BackupRestore.LibraryRestoreBlob), backup)
	if err != nil {
		td.BackupRestore.HasError.LibraryRestoreBlob = true
		td.BackupRestore.Message.LibraryRestoreBlob = "Failed to parse: " + err.Error()
	}

	if !td.BackupRestore.HasError.LibraryRestoreBlob {
		params := &stremio_api.UpdateLibraryItemsParams{Changes: *backup}
		params.APIKey = cookie.AuthKey()
		result, err := client.UpdateLibraryItems(params)
		if err != nil {
			td.BackupRestore.HasError.LibraryRestoreBlob = true
			td.BackupRestore.Message.LibraryRestoreBlob = "Failed to restore: " + err.Error()
		} else if !result.Data.Success {
			td.BackupRestore.HasError.LibraryRestoreBlob = true
			td.BackupRestore.Message.LibraryRestoreBlob = "Failed to restore!"
		} else {
			td.BackupRestore.HasError.LibraryRestoreBlob = false
			td.BackupRestore.Message.LibraryRestoreBlob = "Successfully Restored"
			td.BackupRestore.LibraryRestoreBlob = ""
		}
	}

	buf, err := executeTemplate(td, "sidekick_library_section.html")
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "transportUrl")
	})
}

func AddStremioSidekickEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/configure", withCors(handleConfigure))

	router.HandleFunc("/login", handleLogin)
	router.HandleFunc("/logout", handleLogout)

	router.HandleFunc("/addons", handleAddons)
	router.HandleFunc("/addons/backup", handleAddonsBackup)
	router.HandleFunc("/addons/restore", handleAddonsRestore)
	router.HandleFunc("/addons/reset", handleAddonsReset)
	router.HandleFunc("/addons/{transportUrl}", handleAddonUninstall)
	router.HandleFunc("/addons/{transportUrl}/move/{direction}", handleAddonMove)
	router.HandleFunc("/addons/{transportUrl}/reload", handleAddonReload)
	router.HandleFunc("/addons/{transportUrl}/toggle", handleAddonToggle)
	router.HandleFunc("/addons/{transportUrl}/modify", handleAddonModify)

	router.HandleFunc("/library/backup", handleLibraryBackup)
	router.HandleFunc("/library/restore", handleLibraryRestore)

	mux.Handle("/stremio/sidekick/", http.StripPrefix("/stremio/sidekick", commonMiddleware(router)))
}
