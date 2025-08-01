package endpoint

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
)

//go:embed auth_callback.html
var authCallbackTemplateBlob string

type authCallbackTemplateDataSection struct {
	Title   string        `json:"title"`
	Content template.HTML `json:"content"`
}

type AuthCallbackTemplateData struct {
	Title   string
	Version string
	Code    string
	Error   string

	Provider string
}

var ExecuteAuthCallbackTemplate = func() func(data *AuthCallbackTemplateData) (bytes.Buffer, error) {
	tmpl := template.Must(template.New("auth_callback.html").Parse(authCallbackTemplateBlob))
	return func(data *AuthCallbackTemplateData) (bytes.Buffer, error) {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		return buf, err
	}
}()

func handleTraktAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	td := &AuthCallbackTemplateData{
		Title:    "StremThru",
		Version:  config.Version,
		Provider: "Trakt.tv",
	}

	tok, err := oauth.TraktOAuthConfig.Exchange(code, state)
	if err != nil {
		td.Error = err.Error()
	} else {
		td.Code = tok.Extra("id").(string)
	}

	buf, err := ExecuteAuthCallbackTemplate(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleTMDBAuthInit(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	state := r.URL.Query().Get("state")

	authCodeUrl, err := oauth.TMDBOAuthConfig.TryAuthCodeURL(state)
	if err != nil {
		SendError(w, r, err)
		return
	}

	http.Redirect(w, r, authCodeUrl, http.StatusFound)
}

func handleTMDBAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	td := &AuthCallbackTemplateData{
		Title:    "StremThru",
		Version:  config.Version,
		Provider: "TMDB",
	}

	tok, err := oauth.TMDBOAuthConfig.Exchange(code, state)
	if err != nil {
		td.Error = err.Error()
	} else {
		td.Code = tok.Extra("id").(string)
	}

	buf, err := ExecuteAuthCallbackTemplate(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func handleTVDBAuthToken(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	response_status := 200
	response := struct {
		AccessToken      string `json:"access_token,omitempty"`
		TokenType        string `json:"token_type,omitempty"`
		RefreshToken     string `json:"refresh_token,omitempty"`
		CreatedAt        int64  `json:"created_at,omitempty"`
		ExpiresIn        int    `json:"expires_in,omitempty"`
		ErrorCode        string `json:"error,omitempty"`
		ErrorDescription string `json:"error_description,omitempty"`
		ErrorURI         string `json:"error_uri,omitempty"`
	}{}

	grant_type := r.FormValue("grant_type")
	switch grant_type {
	case "password":
		password := r.FormValue("password")
		if password != config.Integration.TVDB.APIKey {
			response.ErrorCode = "invalid_grant"
			response.ErrorDescription = "Invalid password"
			response_status = 401
		}
	case "refresh_token":
		refresh_token := r.FormValue("refresh_token")
		if refresh_token != config.Integration.TVDB.APIKey {
			response.ErrorCode = "unauthorized_client"
			response.ErrorDescription = "Invalid refresh token"
			response_status = 401
		}
	default:
		response.ErrorCode = "unsupported_grant_type"
		response.ErrorDescription = "Unsupported grant type: " + grant_type
		response_status = 400
	}

	if response_status == 200 {
		res, err := tvdb.NewAPIClient(&tvdb.APIClientConfig{}).Login(&tvdb.LoginParams{
			APIKey: config.Integration.TVDB.APIKey,
		})
		if err != nil {
			response.ErrorCode = "server_error"
			response.ErrorDescription = "Failed to login to TVDB: " + err.Error()
			response_status = 500
		} else {
			response.AccessToken = res.Data.Token
			response.TokenType = "Bearer"
			response.RefreshToken = config.Integration.TVDB.APIKey
			response.ExpiresIn = int(time.Duration(20 * 24 * time.Hour).Seconds())
			response.CreatedAt = time.Now().Unix()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response_status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func AddAuthEndpoints(mux *http.ServeMux) {
	if config.Integration.Trakt.IsEnabled() {
		mux.HandleFunc("/auth/trakt.tv/callback", handleTraktAuthCallback)
	}
	if config.Integration.TMDB.IsEnabled() {
		mux.HandleFunc("/auth/themoviedb.org/init", handleTMDBAuthInit)
		mux.HandleFunc("/auth/themoviedb.org/callback", handleTMDBAuthCallback)
	}
	if config.Integration.TVDB.IsEnabled() {
		mux.HandleFunc("/auth/thetvdb.com/token", handleTVDBAuthToken)
	}
}
