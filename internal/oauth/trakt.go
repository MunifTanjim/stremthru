package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type traktResponseError struct {
	Err     string `json:"error"`
	ErrDesc string `json:"error_description"`
}

func (e *traktResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *traktResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	case strings.Contains(contentType, "text/html"):
		if res.StatusCode >= http.StatusBadRequest {
			errMsg := strings.TrimSpace(string(body))
			if errMsg == "" {
				errMsg = res.Status
			}
			return errors.New(errMsg)
		}
		fallthrough
	default:
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
}

func (r *traktResponseError) GetError(res *http.Response) error {
	if r == nil || r.Err == "" {
		return nil
	}
	return r
}

var TraktTokenSourceConfig = TokenSourceConfig{
	Provider: ProviderTraktTv,
	GetUser: func(client *http.Client, oauthConfig *oauth2.Config) (userId, userName string, err error) {
		req, err := http.NewRequest("GET", "https://api.trakt.tv/users/settings", nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Trakt-API-Key", oauthConfig.ClientID)
		req.Header.Set("Trakt-API-Version", "2")
		res, err := client.Do(req)
		var response struct {
			traktResponseError
			User struct {
				Username string `json:"username"`
				Ids      struct {
					Slug string `json:"slug"`
				} `json:"ids"`
			} `json:"user"`
		}
		err = request.ProcessResponseBody(res, err, &response)
		if err != nil {
			return "", "", err
		}

		return response.User.Ids.Slug, response.User.Username, nil
	},
	PrepareToken: func(tok *oauth2.Token, id, userId string, userName string) *oauth2.Token {
		return tok.WithExtra(map[string]any{
			"id":         id,
			"provider":   ProviderTraktTv,
			"user_id":    userId,
			"user_name":  userName,
			"scope":      tok.Extra("scope").(string),
			"created_at": time.Unix(int64(tok.Extra("created_at").(float64)), 0),
		})
	},
}

var traktOAuthConfig = oauth2.Config{
	ClientID:     config.Integration.Trakt.ClientId,
	ClientSecret: config.Integration.Trakt.ClientSecret,
	Endpoint: oauth2.Endpoint{
		AuthURL:       "https://trakt.tv/oauth/authorize",
		TokenURL:      "https://api.trakt.tv/oauth/token",
		DeviceAuthURL: "https://api.trakt.tv/oauth/device/code",
	},
	RedirectURL: config.BaseURL.JoinPath("/auth/trakt.tv/callback").String(),
}

var TraktOAuthConfig = OAuthConfig{
	Config:      traktOAuthConfig,
	AuthCodeURL: traktOAuthConfig.AuthCodeURL,
	Exchange: func(code, state string) (*oauth2.Token, error) {
		tok, err := traktOAuthConfig.Exchange(context.Background(), code, oauth2.SetAuthURLParam("state", state))
		if err != nil {
			return nil, err
		}

		traktLog.Debug("fetching user info for new token")
		userId, userName, err := TraktTokenSourceConfig.GetUser(
			oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(tok)),
			&traktOAuthConfig,
		)
		if err != nil {
			return nil, err
		}

		existingOTok, err := GetOAuthTokenByUserId(TraktTokenSourceConfig.Provider, userId)
		if err != nil {
			return nil, err
		}

		if existingOTok != nil {
			client := oauth2.NewClient(
				context.Background(),
				DatabaseTokenSource(&DatabaseTokenSourceConfig{
					OAuth:             &traktOAuthConfig,
					TokenSourceConfig: TraktTokenSourceConfig,
				}, existingOTok.ToToken()),
			)

			traktLog.Debug("fetching user info for existing token")
			uId, _, err := TraktTokenSourceConfig.GetUser(
				client,
				&traktOAuthConfig,
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

		tok = TraktTokenSourceConfig.PrepareToken(tok, tokenId, userId, userName)

		otok := &OAuthToken{}
		otok = otok.FromToken(tok)
		err = SaveOAuthToken(otok)
		if err != nil {
			return nil, err
		}

		return tok, nil
	},
}
