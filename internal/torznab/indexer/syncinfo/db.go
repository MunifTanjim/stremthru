package torznab_indexer_syncinfo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
)

var queueCache = cache.NewLRUCache[time.Time](&cache.CacheConfig{
	Lifetime:      3 * time.Hour,
	Name:          "torznab_indexer_syncinfo:queue",
	LocalCapacity: 2048,
})

var staleTime = 24 * time.Hour

type TorznabIndexerSyncInfo struct {
	Type        torznab_indexer.IndexerType `json:"type"`
	Id          string                      `json:"id"`
	SId         string                      `json:"sid"`
	QueuedAt    db.Timestamp                `json:"queued_at"`
	SyncedAt    db.Timestamp                `json:"synced_at"`
	Error       db.NullString               `json:"error"`
	ResultCount sql.NullInt64               `json:"result_count"`
}

func (si *TorznabIndexerSyncInfo) ShouldSync() bool {
	if si.SyncedAt.IsZero() {
		return true
	}
	if !si.QueuedAt.IsZero() && !si.QueuedAt.After(si.SyncedAt.Time) {
		return false
	}
	return si.SyncedAt.Time.Add(staleTime).Before(time.Now())
}

const TableName = "torznab_indexer_syncinfo"

type ColumnStruct struct {
	Type        string
	Id          string
	SId         string
	QueuedAt    string
	SyncedAt    string
	Error       string
	ResultCount string
}

var Column = ColumnStruct{
	Type:        "type",
	Id:          "id",
	SId:         "sid",
	QueuedAt:    "queued_at",
	SyncedAt:    "synced_at",
	Error:       "error",
	ResultCount: "result_count",
}

var columns = []string{
	Column.Type,
	Column.Id,
	Column.SId,
	Column.QueuedAt,
	Column.SyncedAt,
	Column.Error,
	Column.ResultCount,
}

var query_queue = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES (?,?,?,%s) ON CONFLICT (%s) DO UPDATE SET %s",
	TableName,
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
		Column.QueuedAt,
	}, ", "),
	db.CurrentTimestamp,
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
	}, ", "),
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", Column.QueuedAt, Column.QueuedAt),
	}, ","),
)

func Queue(indexerType torznab_indexer.IndexerType, indexerId, sid string) error {
	if sid == "" {
		return nil
	}

	cacheKey := string(indexerType) + ":" + indexerId + ":" + sid

	// Check cache to avoid unnecessary DB writes
	var queuedAt db.Timestamp
	if queueCache.Get(cacheKey, &queuedAt.Time) {
		// Already queued recently
		return nil
	}

	_, err := db.Exec(query_queue, indexerType, indexerId, sid)
	if err == nil {
		err = queueCache.Add(cacheKey, time.Now())
	}
	return err
}

var query_mark_synced = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES (?,?,?,NULL,%s,?) ON CONFLICT (%s) DO UPDATE SET %s",
	TableName,
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
		Column.QueuedAt,
		Column.SyncedAt,
		Column.ResultCount,
	}, ", "),
	db.CurrentTimestamp,
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
	}, ", "),
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", Column.SyncedAt, Column.SyncedAt),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.ResultCount, Column.ResultCount),
	}, ", "),
)

func MarkSynced(indexerType torznab_indexer.IndexerType, indexerId, sid string, resultCount int) error {
	if sid == "" {
		return nil
	}

	_, err := db.Exec(query_mark_synced, indexerType, indexerId, sid, resultCount)
	return err
}

var query_set_sync_error = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES (?,?,?,NULL,NULL,?) ON CONFLICT (%s) DO UPDATE SET %s",
	TableName,
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
		Column.QueuedAt,
		Column.SyncedAt,
		Column.Error,
	}, ", "),
	strings.Join([]string{
		Column.Type,
		Column.Id,
		Column.SId,
	}, ", "),
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", Column.Error, Column.Error),
	}, ", "),
)

func SetSyncError(indexerType torznab_indexer.IndexerType, indexerId, sid string, syncError string) error {
	if sid == "" {
		return nil
	}
	_, err := db.Exec(query_set_sync_error, indexerType, indexerId, sid, db.NullString{String: syncError})
	return err
}

var query_get = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ? AND %s = ? AND %s = ?",
	strings.Join(columns, ", "),
	TableName,
	Column.Type,
	Column.Id,
	Column.SId,
)

func ShouldSync(indexerType torznab_indexer.IndexerType, indexerId, sid string) bool {
	item := TorznabIndexerSyncInfo{}
	row := db.QueryRow(query_get, indexerType, indexerId, sid)
	if err := row.Scan(
		&item.Type,
		&item.Id,
		&item.SId,
		&item.QueuedAt,
		&item.SyncedAt,
		&item.Error,
		&item.ResultCount,
	); err != nil {
		if err == sql.ErrNoRows {
			return true
		}
		return false
	}
	return item.ShouldSync()
}

var query_get_pending_cond = fmt.Sprintf(
	"%s IS NOT NULL AND (%s IS NULL OR (%s > %s AND %s <= ?))",
	Column.QueuedAt,
	Column.SyncedAt,
	Column.QueuedAt,
	Column.SyncedAt,
	Column.SyncedAt,
)

var query_has_sync_pending = fmt.Sprintf(
	"SELECT 1 FROM %s WHERE %s LIMIT 1",
	TableName,
	query_get_pending_cond,
)

func HasSyncPending() bool {
	var one int
	err := db.QueryRow(query_has_sync_pending, db.Timestamp{Time: time.Now().Add(-staleTime)}).Scan(&one)
	return err == nil
}

var query_get_sync_pending = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s",
	db.JoinColumnNames(columns...),
	TableName,
	query_get_pending_cond,
)

func GetSyncPending() ([]TorznabIndexerSyncInfo, error) {
	staleTimestamp := time.Now().Add(-staleTime)

	rows, err := db.Query(query_get_sync_pending, db.Timestamp{Time: staleTimestamp})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TorznabIndexerSyncInfo{}
	for rows.Next() {
		item := TorznabIndexerSyncInfo{}
		if err := rows.Scan(&item.Type, &item.Id, &item.SId, &item.QueuedAt, &item.SyncedAt, &item.Error, &item.ResultCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

type GetItemsParams struct {
	Limit  int
	Offset int
	SId    string
}

var query_get_items = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s DESC LIMIT ? OFFSET ?",
	db.JoinColumnNames(columns...),
	TableName,
	Column.QueuedAt,
	Column.QueuedAt,
)

var query_get_items_by_sid = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IS NOT NULL AND %s = ? ORDER BY %s DESC LIMIT ? OFFSET ?",
	db.JoinColumnNames(columns...),
	TableName,
	Column.QueuedAt,
	Column.SId,
	Column.QueuedAt,
)

func GetItems(params GetItemsParams) ([]TorznabIndexerSyncInfo, error) {
	var rows *sql.Rows
	var err error

	if params.SId != "" {
		rows, err = db.Query(query_get_items_by_sid, params.SId, params.Limit, params.Offset)
	} else {
		rows, err = db.Query(query_get_items, params.Limit, params.Offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TorznabIndexerSyncInfo{}
	for rows.Next() {
		item := TorznabIndexerSyncInfo{}
		if err := rows.Scan(&item.Type, &item.Id, &item.SId, &item.QueuedAt, &item.SyncedAt, &item.Error, &item.ResultCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

var query_count_items = fmt.Sprintf(
	"SELECT COUNT(1) FROM %s WHERE %s IS NOT NULL",
	TableName,
	Column.QueuedAt,
)

var query_count_items_by_sid = fmt.Sprintf(
	"SELECT COUNT(1) FROM %s WHERE %s IS NOT NULL AND %s = ?",
	TableName,
	Column.QueuedAt,
	Column.SId,
)

func CountItems(sid string) (int, error) {
	var count int
	var err error

	if sid != "" {
		err = db.QueryRow(query_count_items_by_sid, sid).Scan(&count)
	} else {
		err = db.QueryRow(query_count_items).Scan(&count)
	}

	if err != nil {
		return 0, err
	}

	return count, nil
}
