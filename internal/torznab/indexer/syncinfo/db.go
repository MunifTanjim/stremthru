package torznab_indexer_syncinfo

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
)

var queueCache = cache.NewLRUCache[time.Time](&cache.CacheConfig{
	Lifetime: 3 * time.Hour,
	Name:     "torznab_indexer_syncinfo:queue",
	MaxSize:  4096,
})

const staleTime = 24 * time.Hour

type Status string

const (
	StatusQueued  Status = "queued"
	StatusSyncing Status = "syncing"
	StatusSynced  Status = "synced"
)

type Query struct {
	Query string `json:"query"` // url.Values.Encode() result
	Done  bool   `json:"done,omitempty"`
	Count int    `json:"count,omitempty"`
	Error string `json:"error,omitempty"`
	Exact bool   `json:"-"`
}

type Queries []Query

func (q Queries) Value() (driver.Value, error) {
	return db.JSONValue(q)
}

func (q *Queries) Scan(value any) error {
	return db.JSONScan(value, q)
}

func (q Queries) Clean() {
	for i := range q {
		item := &q[i]
		item.Done = false
		item.Count = 0
	}
}

func (q Queries) GetProgress() (status Status, count int) {
	allDone, someDone := true, false
	for i := range q {
		if q[i].Done {
			someDone = true
		} else {
			allDone = false
		}
		count += q[i].Count
	}
	if allDone {
		status = StatusSynced
	} else if someDone {
		status = StatusSyncing
	} else {
		status = StatusQueued
	}
	return status, count
}

func (q Queries) GetError() string {
	var str strings.Builder
	for i := range q {
		err := q[i].Error
		if err != "" {
			str.WriteString(strconv.Itoa(i))
			str.WriteString(": ")
			str.WriteString(err)
			str.WriteByte('\n')
		}
	}
	return strings.TrimRight(str.String(), "\n")
}

type TorznabIndexerSyncInfo struct {
	IndexerId   int64         `json:"indexer_id"`
	SId         string        `json:"sid"`
	QueuedAt    db.Timestamp  `json:"queued_at"`
	SyncedAt    db.Timestamp  `json:"synced_at"`
	Error       db.NullString `json:"error"`
	ResultCount sql.NullInt64 `json:"result_count"`
	Status      Status        `json:"status"`
	Queries     Queries       `json:"queries"`
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
	IndexerId   string
	SId         string
	QueuedAt    string
	SyncedAt    string
	Error       string
	ResultCount string
	Status      string
	Queries     string
}

var Column = ColumnStruct{
	IndexerId:   "indexer_id",
	SId:         "sid",
	QueuedAt:    "queued_at",
	SyncedAt:    "synced_at",
	Error:       "error",
	ResultCount: "result_count",
	Status:      "status",
	Queries:     "queries",
}

var columns = []string{
	Column.IndexerId,
	Column.SId,
	Column.QueuedAt,
	Column.SyncedAt,
	Column.Error,
	Column.ResultCount,
	Column.Status,
	Column.Queries,
}

var query_queue = fmt.Sprintf(
	"INSERT INTO %s AS tisi (%s) VALUES (?,?,%s,'%s',?) ON CONFLICT (%s) DO UPDATE SET %s WHERE %s",
	TableName,
	strings.Join([]string{
		Column.IndexerId,
		Column.SId,
		Column.QueuedAt,
		Column.Status,
		Column.Queries,
	}, ", "),
	db.CurrentTimestamp,
	StatusQueued,
	strings.Join([]string{
		Column.IndexerId,
		Column.SId,
	}, ", "),
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", Column.QueuedAt, Column.QueuedAt),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.Status, Column.Status),
		fmt.Sprintf("%s = EXCLUDED.%s", Column.Queries, Column.Queries),
	}, ", "),
	fmt.Sprintf(
		`tisi.%s = '%s'`,
		Column.Status,
		StatusSynced,
	),
)

func Queue(indexerId int64, sid string, queries Queries) error {
	if sid == "" {
		return nil
	}

	cacheKey := strconv.FormatInt(indexerId, 10) + ":" + sid

	var queuedAt db.Timestamp
	if queueCache.Get(cacheKey, &queuedAt.Time) {
		return nil
	}

	queries.Clean()
	_, err := db.Exec(query_queue, indexerId, sid, queries)
	if err == nil {
		err = queueCache.Add(cacheKey, time.Now())
	}
	return err
}

var query_record_progress_common_values = strings.Join([]string{
	fmt.Sprintf("%s = ?", Column.Error),
	fmt.Sprintf("%s = ?", Column.ResultCount),
	fmt.Sprintf("%s = ?", Column.Status),
	fmt.Sprintf("%s = ?", Column.Queries),
}, ", ")

var query_record_progress_where = strings.Join([]string{
	fmt.Sprintf("%s = ?", Column.IndexerId),
	fmt.Sprintf("%s = ?", Column.SId),
}, " AND ")

var query_record_progress = fmt.Sprintf(
	"UPDATE %s SET %s WHERE %s",
	TableName,
	query_record_progress_common_values,
	query_record_progress_where,
)

var query_record_progress_synced = fmt.Sprintf(
	"UPDATE %s SET %s WHERE %s",
	TableName,
	strings.Join([]string{
		query_record_progress_common_values,
		fmt.Sprintf("%s = %s", Column.SyncedAt, db.CurrentTimestamp),
	}, ", "),
	query_record_progress_where,
)

func RecordProgress(indexerId int64, sid string, queries Queries) error {
	if sid == "" {
		return nil
	}
	status, count := queries.GetProgress()
	query := query_record_progress
	if status == StatusSynced {
		query = query_record_progress_synced
	}
	_, err := db.Exec(query,
		db.NullString{String: queries.GetError()},
		sql.NullInt64{Int64: int64(count), Valid: status != StatusQueued},
		status,
		queries,
		indexerId, sid,
	)
	return err
}

var query_get = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ? AND %s = ?",
	strings.Join(columns, ", "),
	TableName,
	Column.IndexerId,
	Column.SId,
)

func ShouldSync(indexerId int64, sid string) bool {
	item := TorznabIndexerSyncInfo{}
	row := db.QueryRow(query_get, indexerId, sid)
	if err := row.Scan(
		&item.IndexerId,
		&item.SId,
		&item.QueuedAt,
		&item.SyncedAt,
		&item.Error,
		&item.ResultCount,
		&item.Status,
		&item.Queries,
	); err != nil {
		if err == sql.ErrNoRows {
			return true
		}
		return false
	}
	return item.ShouldSync()
}

var query_get_pending_cond = fmt.Sprintf(
	// status is pending and (not synced or (queued after synced and synced is stale))
	"%s IN ('%s', '%s') AND (%s IS NULL OR (%s > %s AND %s <= ?))",
	Column.Status, StatusQueued, StatusSyncing,
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
	staleTimestamp := time.Now().Add(-staleTime)
	var one int
	err := db.QueryRow(query_has_sync_pending, db.Timestamp{Time: staleTimestamp}).Scan(&one)
	return err == nil
}

var query_get_sync_pending_by_indexer = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s AND %s = ? ORDER BY %s DESC LIMIT 1000",
	db.JoinColumnNames(columns...),
	TableName,
	query_get_pending_cond,
	Column.IndexerId,
	Column.QueuedAt,
)

func GetSyncPendingByIndexer(indexerId int64) ([]TorznabIndexerSyncInfo, error) {
	staleTimestamp := time.Now().Add(-staleTime)

	rows, err := db.Query(query_get_sync_pending_by_indexer, db.Timestamp{Time: staleTimestamp}, indexerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TorznabIndexerSyncInfo{}
	for rows.Next() {
		item := TorznabIndexerSyncInfo{}
		if err := rows.Scan(&item.IndexerId, &item.SId, &item.QueuedAt, &item.SyncedAt, &item.Error, &item.ResultCount, &item.Status, &item.Queries); err != nil {
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
	"SELECT %s FROM %s ORDER BY %s DESC LIMIT ? OFFSET ?",
	db.JoinColumnNames(columns...),
	TableName,
	Column.QueuedAt,
)

var query_get_items_by_sid = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ? ORDER BY %s DESC LIMIT ? OFFSET ?",
	db.JoinColumnNames(columns...),
	TableName,
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
		if err := rows.Scan(&item.IndexerId, &item.SId, &item.QueuedAt, &item.SyncedAt, &item.Error, &item.ResultCount, &item.Status, &item.Queries); err != nil {
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
	"SELECT COUNT(1) FROM %s",
	TableName,
)

var query_count_items_by_sid = fmt.Sprintf(
	"SELECT COUNT(1) FROM %s WHERE %s = ?",
	TableName,
	Column.SId,
)

type IndexerSyncStats struct {
	IndexerId    int64  `json:"indexer_id"`
	IndexerName  string `json:"indexer_name"`
	TotalCount   int64  `json:"total_count"`
	SyncedCount  int64  `json:"synced_count"`
	QueuedCount  int64  `json:"queued_count"`
	ErrorCount   int64  `json:"error_count"`
	ResultCount  int64  `json:"result_count"`
	LastSyncedAt string `json:"last_synced_at,omitempty"`
}

var query_get_stats = fmt.Sprintf(
	`SELECT
		si.%s,
		i.%s,
		COUNT(1) AS total_count,
		SUM(CASE WHEN si.%s IS NOT NULL THEN 1 ELSE 0 END) AS synced_count,
		SUM(CASE WHEN si.%s IN ('%s', '%s') THEN 1 ELSE 0 END) AS queued_count,
		SUM(CASE WHEN si.%s IS NOT NULL AND si.%s != '' THEN 1 ELSE 0 END) AS error_count,
		COALESCE(SUM(si.%s), 0) AS result_count,
		MAX(si.%s) AS last_synced_at
	FROM %s si
	JOIN %s i ON i.%s = si.%s
	GROUP BY si.%s
	ORDER BY i.%s`,
	Column.IndexerId,
	indexer.Column.Name,
	Column.SyncedAt,
	Column.Status, StatusQueued, StatusSyncing,
	Column.Error, Column.Error,
	Column.ResultCount,
	Column.SyncedAt,
	TableName,
	indexer.TableName, indexer.Column.Id, Column.IndexerId,
	Column.IndexerId,
	indexer.Column.Name,
)

func GetStats() ([]IndexerSyncStats, error) {
	rows, err := db.Query(query_get_stats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []IndexerSyncStats{}
	for rows.Next() {
		item := IndexerSyncStats{}
		lastSyncedAt := db.Timestamp{}
		if err := rows.Scan(
			&item.IndexerId,
			&item.IndexerName,
			&item.TotalCount,
			&item.SyncedCount,
			&item.QueuedCount,
			&item.ErrorCount,
			&item.ResultCount,
			&lastSyncedAt,
		); err != nil {
			return nil, err
		}
		if !lastSyncedAt.IsZero() {
			item.LastSyncedAt = lastSyncedAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

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
