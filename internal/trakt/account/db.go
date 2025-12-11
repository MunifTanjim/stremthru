package trakt_account

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/sync/stremio_trakt"
)

const TableName = "trakt_account"

type TraktAccount struct {
	Id           string
	OAuthTokenId string
	CAt          db.Timestamp
	UAt          db.Timestamp

	otok *oauth.OAuthToken
}

func (a *TraktAccount) OAuthToken() *oauth.OAuthToken {
	if a.otok == nil {
		otok, err := oauth.GetOAuthTokenById(a.OAuthTokenId)
		if err != nil || otok == nil {
			return nil
		}
		a.otok = otok
	}
	return a.otok
}

func (a *TraktAccount) IsValid() bool {
	otok := a.OAuthToken()
	if otok == nil {
		return false
	}
	return !otok.IsExpired()
}

var Column = struct {
	Id           string
	OAuthTokenId string
	CAt          string
	UAt          string
}{
	Id:           "id",
	OAuthTokenId: "oauth_token_id",
	CAt:          "cat",
	UAt:          "uat",
}

var columns = []string{
	Column.Id,
	Column.OAuthTokenId,
	Column.CAt,
	Column.UAt,
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s`,
	strings.Join(columns, ", "),
	TableName,
)

func GetAll() ([]TraktAccount, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TraktAccount{}
	for rows.Next() {
		item := TraktAccount{}
		if err := rows.Scan(&item.Id, &item.OAuthTokenId, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Id,
)

func GetById(id string) (*TraktAccount, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := TraktAccount{}
	if err := row.Scan(&item.Id, &item.OAuthTokenId, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?)`,
	TableName,
	db.JoinColumnNames(
		Column.Id,
		Column.OAuthTokenId,
	),
)

func Insert(oauthTokenId string) (*TraktAccount, error) {
	otok, err := oauth.GetOAuthTokenById(oauthTokenId)
	if err != nil {
		return nil, err
	}
	if otok == nil {
		return nil, errors.New("oauth token not found")
	}
	if otok.Provider != oauth.ProviderTraktTv {
		return nil, errors.New("oauth token is not for trakt.tv")
	}

	id := otok.UserId

	existing, err := GetById(id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	_, err = db.Exec(query_insert, id, oauthTokenId)
	if err != nil {
		return nil, err
	}

	return &TraktAccount{
		Id:           id,
		OAuthTokenId: oauthTokenId,
		CAt:          db.Timestamp{Time: time.Now()},
		UAt:          db.Timestamp{Time: time.Now()},
		otok:         otok,
	}, nil
}

var query_delete = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.Id,
)

func Delete(id string) error {
	if _, err := db.Exec(query_delete, id); err != nil {
		return err
	}
	if err := sync_stremio_trakt.UnlinkByTraktAccount(id); err != nil {
		return err
	}
	return nil
}
