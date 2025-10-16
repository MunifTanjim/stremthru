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
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var letterboxdLog = logger.Scoped("oauth/letterboxd")

type letterboxdResponseError struct {
	Err string `json:"error"`
}

func (e *letterboxdResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *letterboxdResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	default:
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
}

func (r *letterboxdResponseError) GetError(res *http.Response) error {
	if r == nil || r.Err == "" {
		return nil
	}
	return r
}

var LetterboxdTokenSourceConfig = TokenSourceConfig{
	Provider: ProviderLetterboxd,
	GetUser: func(client *http.Client, oauthConfig *oauth2.Config) (userId, userName string, err error) {
		return "", "", errors.New("not implemented")
	},
	PrepareToken: func(tok *oauth2.Token, id, userId, userName string) *oauth2.Token {
		created_at := tok.Extra("created_at")
		if created_at != nil {
			if cat, ok := created_at.(float64); ok {
				created_at = int64(cat)
			} else {
				created_at = nil
			}
		}
		if created_at == nil {
			created_at = time.Now().Unix()
		}
		return tok.WithExtra(map[string]any{
			"id":         id,
			"provider":   ProviderLetterboxd,
			"user_id":    userId,
			"user_name":  userName,
			"scope":      "",
			"created_at": time.Unix(created_at.(int64), 0),
		})
	},
}

var letterboxdOAuthConfig = oauth2.Config{
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://api.letterboxd.com/api/v0/auth/authorize",
		TokenURL:  "https://api.letterboxd.com/api/v0/auth/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
}

var LetterboxdOAuthConfig = OAuthConfig{
	Config: letterboxdOAuthConfig,
	ClientCredentialsToken: func(clientId, clientSecret string) (*oauth2.Token, error) {
		// db level advisory lock to prevent race condition in multi-node deployment
		if lock := db.NewAdvisoryLock("oauth", "token:client-credentials", clientId); lock == nil {
			return nil, errors.New("failed to create advisory lock")
		} else if !lock.Acquire() {
			return nil, errors.New("failed to acquire advisory lock")
		} else {
			defer lock.Release()
		}

		existingOTok, err := GetOAuthTokenByUserId(LetterboxdTokenSourceConfig.Provider, clientId)
		if err != nil {
			return nil, err
		}

		tokenId := uuid.NewString()
		if existingOTok != nil {
			if tok := existingOTok.ToToken(); tok.Valid() {
				letterboxdLog.Debug("existing client credentials token is still valid, reusing it", "client_id", clientId, "token_id", existingOTok.Id, "expiry", tok.Expiry)
				return tok, nil
			}
			tokenId = existingOTok.Id
		}

		letterboxdLog.Debug("fetching new client credentials token", "client_id", clientId, "token_id", tokenId)

		cc_config := clientcredentials.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			TokenURL:     letterboxdOAuthConfig.Endpoint.TokenURL,
			AuthStyle:    letterboxdOAuthConfig.Endpoint.AuthStyle,
		}

		tok, err := cc_config.Token(context.Background())
		if err != nil {
			return nil, err
		}

		tok = LetterboxdTokenSourceConfig.PrepareToken(tok, tokenId, clientId, "")

		otok := &OAuthToken{}
		otok = otok.FromToken(tok)
		err = SaveOAuthToken(otok)
		if err != nil {
			return nil, err
		}

		return tok, nil
	},
}
