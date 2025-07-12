package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var tmdbLog = logger.Scoped("oauth/tmdb")

type tmdbResponseError struct {
	StatusMessage string `json:"status_message"`
	StatusCode    int    `json:"status_code"`
	Success       bool   `json:"success"`
}

func (e *tmdbResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *tmdbResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	default:
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
}

func (r *tmdbResponseError) GetError(res *http.Response) error {
	if r == nil || r.Success || r.StatusCode == 0 {
		return nil
	}
	return r
}

var TMDBTokenSourceConfig = TokenSourceConfig{
	Provider: ProviderTMDB,
	GetUser: func(client *http.Client, oauthConfig *oauth2.Config) (userId, userName string, err error) {
		req, err := http.NewRequest("GET", "https://api.themoviedb.org/3/account", nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Acccept", "application/json")
		res, err := client.Do(req)
		var response struct {
			tmdbResponseError
			Id       int64  `json:"id"`
			Username string `json:"username"`
		}
		err = request.ProcessResponseBody(res, err, &response)
		if err != nil {
			return "", "", err
		}
		return strconv.FormatInt(response.Id, 10), response.Username, nil
	},
	PrepareToken: func(tok *oauth2.Token, id, userId string, userName string) *oauth2.Token {
		return tok.WithExtra(map[string]any{
			"id":         id,
			"provider":   ProviderTMDB,
			"user_id":    userId,
			"user_name":  userName,
			"scope":      tok.Extra("scope").(string),
			"created_at": time.Unix(int64(tok.Extra("created_at").(float64)), 0),
		})
	},
}

var tmdbOAuthConfig = oauth2.Config{
	Endpoint: oauth2.Endpoint{
		AuthURL: config.BaseURL.JoinPath("/auth/themoviedb.org/init").String(),
	},
	RedirectURL: config.BaseURL.JoinPath("/auth/themoviedb.org/callback").String(),
}

var tmdbRequestTokenCache = cache.NewLRUCache[string](&cache.CacheConfig{
	Lifetime: 10 * time.Minute,
	Name:     "oauth:tmdb:request_token",
})

var TMDBOAuthConfig = OAuthConfig{
	Config: tmdbOAuthConfig,
	AuthCodeURL: func(state string, opts ...oauth2.AuthCodeOption) string {
		var buf bytes.Buffer
		buf.WriteString(tmdbOAuthConfig.Endpoint.AuthURL)
		v := url.Values{}
		if state != "" {
			v.Set("state", state)
		}
		if strings.Contains(tmdbOAuthConfig.Endpoint.AuthURL, "?") {
			buf.WriteByte('&')
		} else {
			buf.WriteByte('?')
		}
		buf.WriteString(v.Encode())
		return buf.String()
	},
	TryAuthCodeURL: func(state string, opts ...oauth2.AuthCodeOption) (string, error) {
		code := uuid.NewString()
		redirectTo := tmdbOAuthConfig.RedirectURL + "?code=" + code
		if state != "" {
			redirectTo += "&state=" + state
		}
		jsonBytes, err := json.Marshal(struct {
			RedirectTo string `json:"redirect_to"`
		}{RedirectTo: redirectTo})
		if err != nil {
			return "", err
		}
		body := bytes.NewBuffer(jsonBytes)
		req, err := http.NewRequest("POST", "https://api.themoviedb.org/4/auth/request_token", body)
		if err != nil {
			return "", err
		}
		req.Header.Add("Authorization", "Bearer "+config.Integration.TMDB.AccessToken)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/json")

		res, err := config.DefaultHTTPClient.Do(req)
		var response struct {
			tmdbResponseError
			RequestToken string `json:"request_token"`
		}
		err = request.ProcessResponseBody(res, err, &response)
		if err != nil {
			return "", err
		}

		err = tmdbRequestTokenCache.Add(code+":"+state, response.RequestToken)
		if err != nil {
			return "", err
		}

		return "https://www.themoviedb.org/auth/access?request_token=" + response.RequestToken, nil
	},
	Exchange: func(code, state string) (*oauth2.Token, error) {
		var requestToken string
		if !tmdbRequestTokenCache.Get(code+":"+state, &requestToken) {
			return nil, &oauth2.RetrieveError{
				ErrorCode: "invalid_grant",
			}
		}
		tmdbRequestTokenCache.Remove(code + ":" + state)

		jsonBytes, err := json.Marshal(struct {
			RequestToken string `json:"request_token"`
		}{
			RequestToken: requestToken,
		})
		if err != nil {
			return nil, err
		}
		body := bytes.NewBuffer(jsonBytes)
		req, err := http.NewRequest("POST", "https://api.themoviedb.org/4/auth/access_token", body)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "Bearer "+config.Integration.TMDB.AccessToken)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/json")

		res, err := config.DefaultHTTPClient.Do(req)
		var response struct {
			tmdbResponseError
			AccountId   string `json:"account_id"`
			AccessToken string `json:"access_token"`
		}
		err = request.ProcessResponseBody(res, err, &response)
		if err != nil {
			return nil, err
		}

		tok := &oauth2.Token{AccessToken: response.AccessToken}
		tok = tok.WithExtra(map[string]any{
			"scope":      "",
			"created_at": float64(time.Now().Unix()),
		})

		tmdbLog.Debug("fetching user info for new token")
		userId, userName, err := TMDBTokenSourceConfig.GetUser(
			oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(tok)),
			&tmdbOAuthConfig,
		)
		if err != nil {
			return nil, err
		}

		existingOTok, err := GetOAuthTokenByUserId(TMDBTokenSourceConfig.Provider, userId)
		if err != nil {
			return nil, err
		}

		if existingOTok != nil {
			client := oauth2.NewClient(
				context.Background(),
				DatabaseTokenSource(&DatabaseTokenSourceConfig{
					OAuth:             &tmdbOAuthConfig,
					TokenSourceConfig: TMDBTokenSourceConfig,
				}, existingOTok.ToToken()),
			)

			traktLog.Debug("fetching user info for existing token")
			uId, _, err := TMDBTokenSourceConfig.GetUser(
				client,
				&tmdbOAuthConfig,
			)
			if err != nil || uId != userId {
				existingOTok.AccessToken = ""
				existingOTok.RefreshToken = ""
				err = SaveOAuthToken(existingOTok)
				if err != nil {
					return nil, err
				}
				existingOTok = nil
			}
		}

		tokenId := uuid.NewString()
		if existingOTok != nil {
			tokenId = existingOTok.Id
		}

		tok = TMDBTokenSourceConfig.PrepareToken(tok, tokenId, userId, userName)

		otok := &OAuthToken{}
		otok = otok.FromToken(tok)
		err = SaveOAuthToken(otok)
		if err != nil {
			return nil, err
		}

		return tok, nil
	},
}
