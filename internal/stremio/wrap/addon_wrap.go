package stremio_wrap

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/stremio"
)

var c = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

type UserData struct {
	ManifestURL string   `json:"manifest_url"`
	AuthToken   string   `json:"auth_token"`
	encoded     string   `json:"-"`
	baseUrl     *url.URL `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	return ud.ManifestURL != "" && ud.AuthToken != ""
}

func (ud UserData) GetEncoded() (string, error) {
	if ud.encoded != "" {
		return ud.encoded, nil
	}

	blob, err := json.Marshal(ud)
	if err != nil {
		return "", err
	}
	return core.Base64Encode(string(blob)), nil
}

type userDataError struct {
	manifestUrl string
	authToken   string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	if uderr.manifestUrl != "" {
		str.WriteString("manifest_url: ")
		str.WriteString(uderr.manifestUrl)
		hasSome = true
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.authToken != "" {
		str.WriteString("auth_token: ")
		str.WriteString(uderr.authToken)
	}
	return str.String()
}

func (ud UserData) GetRequestContext(r *http.Request) (*context.RequestContext, error) {
	ctx := &context.RequestContext{}

	authToken := ud.AuthToken
	user, err := core.ParseBasicAuth(authToken)
	if err != nil {
		return ctx, &userDataError{authToken: err.Error()}
	}
	password := config.ProxyAuthPassword.GetPassword(user.Username)
	if password != "" && password == user.Password {
		ctx.IsProxyAuthorized = true
		ctx.ProxyAuthUser = user.Username
		ctx.ProxyAuthPassword = user.Password

		storeName := config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser)
		ctx.Store = shared.GetStore(storeName)
		ctx.StoreAuthToken = config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, storeName)
	}

	if !ctx.IsProxyAuthorized {
		return ctx, &userDataError{authToken: "Invalid Auth Token"}
	}

	if ud.baseUrl == nil {
		return ctx, &userDataError{manifestUrl: "Invalid Manifest URL"}
	}

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}

	if IsMethod(r, http.MethodGet) {
		data.encoded = r.PathValue("userData")
		if data.encoded == "" {
			return data, nil
		}
		blob, err := core.Base64DecodeToByte(data.encoded)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(blob, data)
		if err != nil {
			return nil, err
		}
	}

	if IsMethod(r, http.MethodPost) {
		data.ManifestURL = r.FormValue("manifest_url")
		data.AuthToken = r.FormValue("auth_token")
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	if data.ManifestURL != "" {
		if baseUrl, err := url.Parse(data.ManifestURL); err == nil {
			baseUrl.Path = strings.TrimSuffix(baseUrl.Path, "/manifest.json")
			data.baseUrl = baseUrl
		}
	}

	return data, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/wrap/configure", http.StatusFound)
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	res, err := c.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl})
	if err != nil {
		SendError(w, err)
		return
	}

	manifest := getManifest(&res.Data, ud)

	SendResponse(w, 200, manifest)
}

func getTemplateData() *configure.TemplateData {
	return &configure.TemplateData{
		Title: "StremThru Wrap",
		Configs: []configure.Config{
			configure.Config{
				Key:         "manifest_url",
				Type:        "url",
				Default:     "",
				Title:       "Upstream Manifest URL",
				Description: "Manifest URL for the Upstream Addon",
				Required:    true,
			},
			configure.Config{
				Key:         "auth_token",
				Type:        "password",
				Default:     "",
				Title:       "StremThru Token",
				Description: `StremThru Basic Auth Token (base64) from <a href="https://github.com/MunifTanjim/stremthru?tab=readme-ov-file#configuration" target="_blank"><code>STREMTHRU_PROXY_AUTH</code></a>`,
				Required:    true,
			},
		},
	}
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	td := getTemplateData()
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "manifest_url":
			conf.Default = ud.ManifestURL
		case "auth_token":
			conf.Default = ud.AuthToken
		}
	}

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/manifest.json").String()
			}
		}

		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	var manifest_url_config *configure.Config
	var auth_token_config *configure.Config
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "manifest_url":
			manifest_url_config = conf
		case "auth_token":
			auth_token_config = conf
		}
	}

	_, err = ud.GetRequestContext(r)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			if uderr.manifestUrl != "" {
				manifest_url_config.Error = uderr.manifestUrl
			}
			if uderr.authToken != "" {
				auth_token_config.Error = uderr.authToken
			}
		} else {
			SendError(w, err)
			return
		}
	}

	if manifest_url_config.Error == "" {
		_, err := c.GetManifest(&stremio_addon.GetManifestParams{BaseURL: ud.baseUrl})
		if err != nil {
			manifest_url_config.Error = "Failed to fetch Manifest"
		}
	}

	if td.HasError() {
		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, err)
		return
	}

	url := ExtractRequestBaseURL(r).JoinPath("/stremio/wrap/" + eud + "/configure")
	q := url.Query()
	q.Set("try_install", "1")
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, err)
		return
	}

	resource := r.PathValue("resource")
	contentType := r.PathValue("contentType")
	id := r.PathValue("id")
	extra := r.PathValue("extra")

	if resource == string(stremio.ResourceNameStream) {
		res, err := c.FetchStream(&stremio_addon.FetchStreamParams{
			BaseURL: ud.baseUrl,
			Type:    contentType,
			Id:      id,
			Extra:   extra,
		})
		if err != nil {
			SendError(w, err)
			return
		}

		ctx, err := ud.GetRequestContext(r)
		if err != nil {
			SendError(w, err)
			return
		}

		for i := range res.Data.Streams {
			stream := &res.Data.Streams[i]
			if stream.URL != "" {
				if url, err := shared.CreateProxyLink(r, ctx, stream.URL); err == nil {
					stream.URL = url
					stream.Name = "✨ " + stream.Name
				}
			}
		}

		SendResponse(w, 200, res.Data)
		return
	}

	c.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.baseUrl,
		Resource: resource,
		Type:     contentType,
		Id:       id,
		Extra:    extra,
	})
}

func AddStremioWrapEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/stremio/wrap", handleRoot)
	mux.HandleFunc("/stremio/wrap/{$}", handleRoot)

	mux.HandleFunc("/stremio/wrap/manifest.json", handleManifest)
	mux.HandleFunc("/stremio/wrap/{userData}/manifest.json", handleManifest)

	mux.HandleFunc("/stremio/wrap/configure", handleConfigure)
	mux.HandleFunc("/stremio/wrap/{userData}/configure", handleConfigure)

	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}", handleResource)
	mux.HandleFunc("/stremio/wrap/{userData}/{resource}/{contentType}/{id}/{extra}", handleResource)
}
