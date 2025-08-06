package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/request"
	"golang.org/x/oauth2"
)

var tvdbLog = logger.Scoped("oauth/tvdb")

type tvdbResponseError struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func (e *tvdbResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *tvdbResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	default:
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
}

func (r *tvdbResponseError) GetError(res *http.Response) error {
	if r == nil || r.Status == "success" {
		return nil
	}
	return r
}

var TVDBTokenSourceConfig = TokenSourceConfig{
	Provider: ProviderTVDB,
	GetUser: func(client *http.Client, oauthConfig *oauth2.Config) (userId, userName string, err error) {
		req, err := http.NewRequest("GET", "https://api4.thetvdb.com/v4/user", nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Accept", "application/json")
		res, err := client.Do(req)
		var response struct {
			tvdbResponseError
			Data struct {
				Id   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		err = request.ProcessResponseBody(res, err, &response)
		if err != nil {
			return "", "", err
		}
		return strconv.FormatInt(response.Data.Id, 10), response.Data.Name, nil
	},
	PrepareToken: func(tok *oauth2.Token, id, userId, userName string) *oauth2.Token {
		return tok.WithExtra(map[string]any{
			"id":         id,
			"provider":   ProviderTVDB,
			"user_id":    userId,
			"user_name":  userName,
			"scope":      "",
			"created_at": time.Unix(int64(tok.Extra("created_at").(float64)), 0),
		})
	},
}

var tvdbOAuthConfig = oauth2.Config{
	Endpoint: oauth2.Endpoint{
		TokenURL: config.BaseURL.JoinPath("/auth/thetvdb.com/token").String(),
	},
}

var TVDBOAuthConfig = OAuthConfig{
	Config: tvdbOAuthConfig,
	PasswordCredentialsToken: func(username, password string) (*oauth2.Token, error) {
		tok, err := tvdbOAuthConfig.PasswordCredentialsToken(context.Background(), username, password)
		if err != nil {
			return nil, err
		}

		tvdbLog.Debug("fetching user info for new token")
		userId, userName, err := TVDBTokenSourceConfig.GetUser(
			oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(tok)),
			&tvdbOAuthConfig,
		)
		if err != nil {
			return nil, err
		}

		tok = TVDBTokenSourceConfig.PrepareToken(tok, config.Integration.TVDB.SystemOAuthTokenId, userId, userName)

		otok := &OAuthToken{}
		otok = otok.FromToken(tok)
		err = SaveOAuthToken(otok)
		if err != nil {
			return nil, err
		}

		return tok, nil
	},
}
