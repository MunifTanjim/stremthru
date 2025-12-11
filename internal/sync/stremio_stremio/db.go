package sync_stremio_stremio

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "sync_stremio_stremio_link"

type SyncDirection string

const (
	SyncDirectionNone SyncDirection = "none"
	SyncDirectionAToB SyncDirection = "a_to_b"
	SyncDirectionBToA SyncDirection = "b_to_a"
	SyncDirectionBoth SyncDirection = "both"
)

func (d SyncDirection) IsValid() bool {
	switch d {
	case SyncDirectionNone, SyncDirectionAToB, SyncDirectionBToA, SyncDirectionBoth:
		return true
	}
	return false
}

func (d SyncDirection) ShouldSyncAToB() bool {
	return d == SyncDirectionAToB || d == SyncDirectionBoth
}

func (d SyncDirection) ShouldSyncBToA() bool {
	return d == SyncDirectionBToA || d == SyncDirectionBoth
}

func (d SyncDirection) IsDisabled() bool {
	return d == SyncDirectionNone
}

type SyncConfigWatched struct {
	Direction SyncDirection `json:"dir"`
	Ids       []string      `json:"ids"`
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

type SyncStremioStremioLink struct {
	AccountAId string
	AccountBId string
	SyncConfig SyncConfig
	SyncState  SyncState
	CAt        db.Timestamp
	UAt        db.Timestamp
}

var Column = struct {
	AccountAId string
	AccountBId string
	SyncConfig string
	SyncState  string
	CAt        string
	UAt        string
}{
	AccountAId: "account_a_id",
	AccountBId: "account_b_id",
	SyncConfig: "sync_config",
	SyncState:  "sync_state",
	CAt:        "cat",
	UAt:        "uat",
}

var columns = []string{
	Column.AccountAId,
	Column.AccountBId,
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

func GetAll() ([]SyncStremioStremioLink, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []SyncStremioStremioLink{}
	for rows.Next() {
		item := SyncStremioStremioLink{}
		if err := rows.Scan(&item.AccountAId, &item.AccountBId, &item.SyncConfig, &item.SyncState, &item.CAt, &item.UAt); err != nil {
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
	Column.AccountAId,
	Column.AccountBId,
)

func GetById(accountAId, accountBId string) (*SyncStremioStremioLink, error) {
	row := db.QueryRow(query_get_by_account_id, accountAId, accountBId)
	item := SyncStremioStremioLink{}
	if err := row.Scan(
		&item.AccountAId,
		&item.AccountBId,
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
		Column.AccountAId,
		Column.AccountBId,
		Column.SyncConfig,
	),
)

func Link(accountAId, accountBId string, syncConfig SyncConfig) (*SyncStremioStremioLink, error) {
	_, err := db.Exec(query_insert, accountAId, accountBId, syncConfig)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &SyncStremioStremioLink{
		AccountAId: accountAId,
		AccountBId: accountBId,
		SyncConfig: syncConfig,
		SyncState:  SyncState{},
		CAt:        db.Timestamp{Time: now},
		UAt:        db.Timestamp{Time: now},
	}, nil
}

var query_unlink = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.AccountAId,
	Column.AccountBId,
)

func Unlink(accountAId, accountBId string) error {
	_, err := db.Exec(query_unlink, accountAId, accountBId)
	return err
}

var query_unlink_by_account_a = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.AccountAId,
)

func UnlinkByAccountA(accountAId string) error {
	_, err := db.Exec(query_unlink_by_account_a, accountAId)
	return err
}

var query_unlink_by_account_b = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.AccountBId,
)

func UnlinkByAccountB(accountBId string) error {
	_, err := db.Exec(query_unlink_by_account_b, accountBId)
	return err
}

func UnlinkByStremioAccount(accountId string) error {
	if err := UnlinkByAccountA(accountId); err != nil {
		return err
	}
	if err := UnlinkByAccountB(accountId); err != nil {
		return err
	}
	return nil
}

var query_set_sync_config = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.SyncConfig,
	Column.UAt, db.CurrentTimestamp,
	Column.AccountAId,
	Column.AccountBId,
)

func SetSyncConfig(accountAId, accountBId string, syncConfig SyncConfig) error {
	_, err := db.Exec(query_set_sync_config, syncConfig, accountAId, accountBId)
	return err
}

var query_set_sync_state = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.SyncState,
	Column.UAt, db.CurrentTimestamp,
	Column.AccountAId,
	Column.AccountBId,
)

func SetSyncState(accountAId, accountBId string, syncState SyncState) error {
	_, err := db.Exec(query_set_sync_state,
		syncState,
		accountAId,
		accountBId,
	)
	return err
}
