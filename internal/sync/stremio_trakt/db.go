package sync_stremio_trakt

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "sync_stremio_trakt_link"

type SyncDirection string

const (
	SyncDirectionNone           SyncDirection = "none"
	SyncDirectionStremioToTrakt SyncDirection = "stremio_to_trakt"
	SyncDirectionTraktToStremio SyncDirection = "trakt_to_stremio"
	SyncDirectionBoth           SyncDirection = "both"
)

func (d SyncDirection) IsValid() bool {
	switch d {
	case SyncDirectionNone, SyncDirectionStremioToTrakt, SyncDirectionTraktToStremio, SyncDirectionBoth:
		return true
	}
	return false
}

func (d SyncDirection) ShouldSyncToTrakt() bool {
	return d == SyncDirectionStremioToTrakt || d == SyncDirectionBoth
}

func (d SyncDirection) ShouldSyncToStremio() bool {
	return d == SyncDirectionTraktToStremio || d == SyncDirectionBoth
}

func (d SyncDirection) IsDisabled() bool {
	return d == SyncDirectionNone
}

type SyncConfigWatched struct {
	Direction SyncDirection `json:"dir"`
}

type SyncConfig struct {
	Watched SyncConfigWatched `json:"watched"`
}

func (sc SyncConfig) Value() (driver.Value, error) {
	return db.JSONValue(sc)
}

func (sc *SyncConfig) Scan(value any) error {
	return db.JSONScan(value, sc)
}

type SyncStateWatched struct {
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

type SyncState struct {
	Watched SyncStateWatched `json:"watched"`
}

func (ss SyncState) Value() (driver.Value, error) {
	return db.JSONValue(ss)
}

func (ss *SyncState) Scan(value any) error {
	return db.JSONScan(value, ss)
}

type SyncStremioTraktLink struct {
	StremioAccountId string
	TraktAccountId   string
	SyncConfig       SyncConfig
	SyncState        SyncState
	CAt              db.Timestamp
	UAt              db.Timestamp
}

var Column = struct {
	StremioAccountId string
	TraktAccountId   string
	SyncConfig       string
	SyncState        string
	CAt              string
	UAt              string
}{
	StremioAccountId: "stremio_account_id",
	TraktAccountId:   "trakt_account_id",
	SyncConfig:       "sync_config",
	SyncState:        "sync_state",
	CAt:              "cat",
	UAt:              "uat",
}

var columns = []string{
	Column.StremioAccountId,
	Column.TraktAccountId,
	Column.SyncConfig,
	Column.SyncState,
	Column.CAt,
	Column.UAt,
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s`,
	strings.Join(columns, ", "),
	TableName,
)

func GetAll() ([]SyncStremioTraktLink, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []SyncStremioTraktLink{}
	for rows.Next() {
		item := SyncStremioTraktLink{}
		if err := rows.Scan(&item.StremioAccountId, &item.TraktAccountId, &item.SyncConfig, &item.SyncState, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

var query_get_by_account_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.StremioAccountId,
	Column.TraktAccountId,
)

func GetById(stremioAccountId, traktAccountId string) (*SyncStremioTraktLink, error) {
	row := db.QueryRow(query_get_by_account_id, stremioAccountId, traktAccountId)
	item := SyncStremioTraktLink{}
	if err := row.Scan(
		&item.StremioAccountId,
		&item.TraktAccountId,
		&item.SyncConfig,
		&item.SyncState,
		&item.CAt,
		&item.UAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?)`,
	TableName,
	db.JoinColumnNames(
		Column.StremioAccountId,
		Column.TraktAccountId,
		Column.SyncConfig,
	),
)

func Link(stremioAccountId, traktAccountId string, syncConfig SyncConfig) (*SyncStremioTraktLink, error) {
	_, err := db.Exec(query_insert, stremioAccountId, traktAccountId, syncConfig)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &SyncStremioTraktLink{
		StremioAccountId: stremioAccountId,
		TraktAccountId:   traktAccountId,
		SyncConfig:       syncConfig,
		SyncState:        SyncState{},
		CAt:              db.Timestamp{Time: now},
		UAt:              db.Timestamp{Time: now},
	}, nil
}

var query_unlink = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.StremioAccountId,
	Column.TraktAccountId,
)

func Unlink(stremioAccountId, traktAccountId string) error {
	_, err := db.Exec(query_unlink, stremioAccountId, traktAccountId)
	return err
}

var query_unlink_by_stremio_account = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.StremioAccountId,
)

func UnlinkByStremioAccount(stremioAccountId string) error {
	_, err := db.Exec(query_unlink_by_stremio_account, stremioAccountId)
	return err
}

var query_unlink_by_trakt_account = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.TraktAccountId,
)

func UnlinkByTraktAccount(traktAccountId string) error {
	_, err := db.Exec(query_unlink_by_trakt_account, traktAccountId)
	return err
}

var query_set_sync_config = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.SyncConfig,
	Column.UAt, db.CurrentTimestamp,
	Column.StremioAccountId,
	Column.TraktAccountId,
)

func SetSyncConfig(stremioAccountId, traktAccontId string, syncConfig SyncConfig) error {
	_, err := db.Exec(query_set_sync_config, syncConfig, stremioAccountId, traktAccontId)
	return err
}

var query_set_sync_state = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.SyncState,
	Column.UAt, db.CurrentTimestamp,
	Column.StremioAccountId,
	Column.TraktAccountId,
)

func SetSyncState(stremioAccountId, traktAccountId string, syncState SyncState) error {
	_, err := db.Exec(query_set_sync_state,
		syncState,
		stremioAccountId,
		traktAccountId,
	)
	return err
}
