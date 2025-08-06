package oauth

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
	"golang.org/x/oauth2"
)

const TableName = "oauth_token"

type Provider string

const (
	ProviderKitsu   Provider = "kitsu.app"
	ProviderTMDB    Provider = "themoviedb.org"
	ProviderTraktTv Provider = "trakt.tv"
	ProviderTVDB    Provider = "thetvdb.com"
)

type OAuthToken struct {
	Id           string
	Provider     Provider
	UserId       string
	UserName     string
	TokenType    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    db.Timestamp
	Scope        db.CommaSeperatedString
	Version      int
	CreatedAt    db.Timestamp
	UpdatedAt    db.Timestamp
}

func (otok *OAuthToken) FromToken(tok *oauth2.Token) *OAuthToken {
	if otok.Id == "" {
		otok.Id = tok.Extra("id").(string)
	}
	if otok.Provider == "" {
		otok.Provider = tok.Extra("provider").(Provider)
	}
	if otok.UserId == "" {
		otok.UserId = tok.Extra("user_id").(string)
	}
	if otok.UserName == "" {
		otok.UserName = tok.Extra("user_name").(string)
	}
	otok.TokenType = tok.TokenType
	otok.AccessToken = tok.AccessToken
	otok.RefreshToken = tok.RefreshToken
	otok.ExpiresAt = db.Timestamp{Time: tok.Expiry}
	if len(otok.Scope) == 0 {
		otok.Scope = strings.Fields(tok.Extra("scope").(string))
	}
	created_at := tok.Extra("created_at").(time.Time)
	if otok.CreatedAt.IsZero() {
		otok.CreatedAt = db.Timestamp{Time: created_at}
	}
	if otok.UpdatedAt.IsZero() {
		otok.UpdatedAt = db.Timestamp{Time: created_at}
	}
	return otok
}

func (otok *OAuthToken) ToToken() *oauth2.Token {
	if otok == nil {
		return nil
	}
	tok := &oauth2.Token{
		TokenType:    otok.TokenType,
		AccessToken:  otok.AccessToken,
		RefreshToken: otok.RefreshToken,
		Expiry:       otok.ExpiresAt.Time,
		ExpiresIn:    int64(time.Until(otok.ExpiresAt.Time).Seconds()),
	}
	return tok.WithExtra(map[string]any{
		"id":         otok.Id,
		"provider":   otok.Provider,
		"user_id":    otok.UserId,
		"user_name":  otok.UserName,
		"scope":      strings.Join(otok.Scope, " "),
		"created_at": otok.CreatedAt.Time,
	})
}

func (otok OAuthToken) IsExpired() bool {
	return otok.ExpiresAt.Before(time.Now())
}

var Column = struct {
	Id           string
	Provider     string
	UserId       string
	UserName     string
	TokenType    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
	Scope        string
	Version      string
	CreatedAt    string
	UpdatedAt    string
}{
	Id:           "id",
	Provider:     "provider",
	UserId:       "user_id",
	UserName:     "user_name",
	TokenType:    "token_type",
	AccessToken:  "access_token",
	RefreshToken: "refresh_token",
	ExpiresAt:    "expires_at",
	Scope:        "scope",
	Version:      "v",
	CreatedAt:    "cat",
	UpdatedAt:    "uat",
}

var columns = []string{
	Column.Id,
	Column.Provider,
	Column.UserId,
	Column.UserName,
	Column.TokenType,
	Column.AccessToken,
	Column.RefreshToken,
	Column.ExpiresAt,
	Column.Scope,
	Column.Version,
	Column.CreatedAt,
	Column.UpdatedAt,
}

var query_get_oauth_token_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Id,
)

func GetOAuthTokenById(id string) (*OAuthToken, error) {
	row := db.QueryRow(query_get_oauth_token_by_id, id)
	otok := OAuthToken{}
	if err := row.Scan(
		&otok.Id,
		&otok.Provider,
		&otok.UserId,
		&otok.UserName,
		&otok.TokenType,
		&otok.AccessToken,
		&otok.RefreshToken,
		&otok.ExpiresAt,
		&otok.Scope,
		&otok.Version,
		&otok.CreatedAt,
		&otok.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("GetOAuthTokenById: not found", "id", id)
			return nil, nil
		}
		return nil, err
	}
	return &otok, nil
}

var query_get_oauth_token_by_user_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Provider,
	Column.UserId,
)

func GetOAuthTokenByUserId(provider Provider, userId string) (*OAuthToken, error) {
	query := query_get_oauth_token_by_user_id
	row := db.QueryRow(query, provider, userId)
	otok := OAuthToken{}
	if err := row.Scan(
		&otok.Id,
		&otok.Provider,
		&otok.UserId,
		&otok.UserName,
		&otok.TokenType,
		&otok.AccessToken,
		&otok.RefreshToken,
		&otok.ExpiresAt,
		&otok.Scope,
		&otok.Version,
		&otok.CreatedAt,
		&otok.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("GetOAuthTokenByUserId: not found", "provider", provider, "user_id", userId)
			return nil, nil
		}
		return nil, err
	}
	return &otok, nil
}

var query_save_oauth_token = fmt.Sprintf(
	`INSERT INTO %s AS ot (%s) VALUES (%s) ON CONFLICT(%s,%s) DO UPDATE SET %s`,
	TableName,
	strings.Join(columns, ","),
	util.RepeatJoin("?", len(columns), ","),
	Column.Provider,
	Column.UserId,
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", Column.UserName, Column.UserName),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.TokenType, Column.TokenType),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.AccessToken, Column.AccessToken),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.RefreshToken, Column.RefreshToken),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.ExpiresAt, Column.ExpiresAt),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.Scope, Column.Scope),
		fmt.Sprintf("%s = ot.%s + 1", Column.Version, Column.Version),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.UpdatedAt, Column.UpdatedAt),
	}, ", "),
)

var query_delete_oauth_token = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.Id,
)

func SaveOAuthToken(otok *OAuthToken) error {
	if otok == nil {
		return nil
	}

	if otok.AccessToken == "" && otok.RefreshToken == "" {
		log.Debug("SaveOAuthToken: deleting token", "id", otok.Id)
		_, err := db.Exec(query_delete_oauth_token, otok.Id)
		return err
	}

	log.Debug("SaveOAuthToken: saving token", "provider", otok.Provider, "user_id", otok.UserId, "user_name", otok.UserName)
	_, err := db.Exec(
		query_save_oauth_token,
		otok.Id,
		otok.Provider,
		otok.UserId,
		otok.UserName,
		otok.TokenType,
		otok.AccessToken,
		otok.RefreshToken,
		otok.ExpiresAt,
		otok.Scope,
		otok.Version,
		otok.CreatedAt,
		otok.UpdatedAt,
	)
	return err
}
