package oauth

import (
	"context"
	"errors"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/db"
	"golang.org/x/oauth2"
)

type TokenSourceConfig struct {
	Provider     Provider
	GetUser      func(client *http.Client, oauthConfig *oauth2.Config) (userId, userName string, err error)
	PrepareToken func(tok *oauth2.Token, id, userId, userName string) *oauth2.Token
}

type dbTokenSource struct {
	oauth2.TokenSource
	oauthConfig *oauth2.Config
	config      TokenSourceConfig
	get_user    func(*oauth2.Token, *TokenSourceConfig, *oauth2.Config) (userId, userName string, err error)
	refresh     func(oauth2.TokenSource) (*oauth2.Token, error)
	TokenId     string
	tok         *oauth2.Token
}

func (ts *dbTokenSource) save() error {
	otok := &OAuthToken{}
	otok = otok.FromToken(ts.tok)
	tokenSourceLog.Debug("saving token", "provider", ts.config.Provider, "user_id", otok.UserId, "user_name", otok.UserName)
	return SaveOAuthToken(otok)
}

func (ts *dbTokenSource) Token() (*oauth2.Token, error) {
	if ts.tok.Valid() {
		return ts.tok, nil
	}

	// db level advisory lock to prevent race condition in multi-node deployment
	if lock := db.NewAdvisoryLock("oauth", "token", ts.TokenId); lock == nil {
		return nil, errors.New("failed to create advisory lock")
	} else if !lock.Acquire() {
		return nil, errors.New("failed to acquire advisory lock")
	} else {
		defer lock.Release()
	}

	// read from db, in case already refreshed token exists
	otok, err := GetOAuthTokenById(ts.TokenId)
	if err != nil {
		return nil, err
	}
	if otok == nil {
		return nil, errors.New("invalid id, token not found")
	}

	ts.tok = otok.ToToken()
	if ts.tok.Valid() {
		return ts.tok, nil
	}

	tokenSourceLog.Debug("token expired, refreshing token", "provider", ts.config.Provider, "user_id", ts.tok.Extra("user_id"), "user_name", ts.tok.Extra("user_name"))
	tok, err := ts.refresh(ts.TokenSource)
	if err != nil {
		tokenSourceLog.Error("failed to refresh token", "error", err, "provider", ts.config.Provider, "user_id", ts.tok.Extra("user_id"), "user_name", ts.tok.Extra("user_name"))
		return nil, err
	}

	userId, userName, err := ts.get_user(tok, &ts.config, ts.oauthConfig)
	if err != nil {
		return nil, err
	}

	ts.tok = ts.config.PrepareToken(tok, ts.TokenId, userId, userName)
	if err := ts.save(); err != nil {
		return nil, err
	}

	return ts.tok, nil
}

type DatabaseTokenSourceConfig struct {
	OAuth *oauth2.Config
	TokenSourceConfig
	GetUser func(*oauth2.Token, *TokenSourceConfig, *oauth2.Config) (userId, userName string, err error)
	Refresh func(oauth2.TokenSource) (*oauth2.Token, error)
}

func defaultTokenSourceRefresher(ts oauth2.TokenSource) (*oauth2.Token, error) {
	return ts.Token()
}

func defaultDBTokenSourceGetUser(tok *oauth2.Token, tsc *TokenSourceConfig, oauthConfig *oauth2.Config) (userId, userName string, err error) {
	return tsc.GetUser(
		oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(tok)),
		oauthConfig,
	)
}

func DatabaseTokenSource(conf *DatabaseTokenSourceConfig, token *oauth2.Token) oauth2.TokenSource {
	if conf.GetUser == nil {
		conf.GetUser = defaultDBTokenSourceGetUser
	}
	if conf.Refresh == nil {
		conf.Refresh = defaultTokenSourceRefresher
	}
	return oauth2.ReuseTokenSource(token, &dbTokenSource{
		TokenSource: conf.OAuth.TokenSource(context.Background(), token),
		config:      conf.TokenSourceConfig,
		get_user:    conf.GetUser,
		refresh:     conf.Refresh,
		oauthConfig: conf.OAuth,
		TokenId:     token.Extra("id").(string),
		tok:         token,
	})
}
