package endpoint

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/shared"
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

func AddAuthEndpoints(mux *http.ServeMux) {
	if config.Integration.Trakt.IsEnabled() {
		mux.HandleFunc("/auth/trakt.tv/callback", handleTraktAuthCallback)
	}
	if config.Integration.TMDB.IsEnabled() {
		mux.HandleFunc("/auth/themoviedb.org/init", handleTMDBAuthInit)
		mux.HandleFunc("/auth/themoviedb.org/callback", handleTMDBAuthCallback)
	}
}
