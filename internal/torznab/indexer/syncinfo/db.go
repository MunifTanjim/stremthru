package torznab_indexer_syncinfo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
)

var log = logger.Scoped("torznab-indexer-syncinfo")

var queueCache = cache.NewLRUCache[time.Time](&cache.CacheConfig{
	Lifetime:      1 * time.Hour,
	Name:          "torznab_indexer_syncinfo:queue",
	LocalCapacity: 8192,
})

type TorznabIndexerSyncInfo struct {
	Type     torznab_indexer.IndexerType `json:"type"`
	Id       string                      `json:"id"`
	SId      string                      `json:"sid"`
	QueuedAt db.Timestamp                `json:"queued_at"`
	SyncedAt db.Timestamp                `json:"synced_at"`
}

const TableName = "torznab_indexer_syncinfo"

type ColumnStruct struct {
	Type     string
	Id       string
	SId      string
	QueuedAt string
	SyncedAt string
}

var Column = ColumnStruct{
	Type:     "type",
	Id:       "id",
	SId:      "sid",
	QueuedAt: "queued_at",
	SyncedAt: "synced_at",
}

var columns = []string{
	Column.Type,
	Column.Id,
	Column.SId,
	Column.QueuedAt,
	Column.SyncedAt,
}

var query_queue = fmt.Sprintf(
	"INSERT INTO %s (%s,%s,%s,%s) VALUES (?,?,?,%s) ON CONFLICT (%s,%s,%s) DO UPDATE SET %s = EXCLUDED.%s",
	TableName,
	Column.Type,
	Column.Id,
	Column.SId,
	Column.QueuedAt,
	db.CurrentTimestamp,
	Column.Type,
	Column.Id,
	Column.SId,
	Column.QueuedAt,
	Column.QueuedAt,
)

func Queue(indexerType torznab_indexer.IndexerType, indexerId, sid string) {
	if sid == "" {
		return
	}

	cacheKey := string(indexerType) + ":" + indexerId + ":" + sid

	// Check cache to avoid unnecessary DB writes
	var queuedAt db.Timestamp
	if queueCache.Get(cacheKey, &queuedAt.Time) {
		// Already queued recently
		return
	}

	_, err := db.Exec(query_queue, indexerType, indexerId, sid)
	if err == nil {
		err = queueCache.Add(cacheKey, time.Now())
	}
	if err != nil {
		log.Error("failed to queue", "error", err, "type", indexerType, "id", indexerId, "sid", sid)
	}
}

var query_mark_synced = fmt.Sprintf(
	"UPDATE %s SET %s = %s WHERE %s = ? AND %s = ? AND %s = ?",
	TableName,
	Column.SyncedAt,
	db.CurrentTimestamp,
	Column.Type,
	Column.Id,
	Column.SId,
)

func MarkSynced(indexerType torznab_indexer.IndexerType, indexerId, sid string) {
	if sid == "" {
		return
	}

	_, err := db.Exec(query_mark_synced, indexerType, indexerId, sid)
	if err != nil {
		log.Error("failed to mark synced", "error", err, "type", indexerType, "id", indexerId, "sid", sid)
	}
}

var staleTime = 24 * time.Hour

var query_has_pending = fmt.Sprintf(
	"SELECT 1 FROM %s WHERE %s IS NULL OR (%s > %s AND %s <= ?) LIMIT 1",
	TableName,
	Column.SyncedAt,
	Column.QueuedAt,
	Column.SyncedAt,
	Column.SyncedAt,
)

func HasPending() bool {
	var one int
	err := db.QueryRow(query_has_pending, time.Now().Add(-staleTime)).Scan(&one)
	return err == nil
}

var query_get_pending = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IS NULL OR (%s > %s AND %s <= ?)",
	db.JoinColumnNames(columns...),
	TableName,
	Column.SyncedAt,
	Column.QueuedAt,
	Column.SyncedAt,
	Column.SyncedAt,
)

func GetPending() ([]TorznabIndexerSyncInfo, error) {
	staleTimestamp := time.Now().Add(-staleTime)

	rows, err := db.Query(query_get_pending, staleTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TorznabIndexerSyncInfo{}
	for rows.Next() {
		item := TorznabIndexerSyncInfo{}
		if err := rows.Scan(&item.Type, &item.Id, &item.SId, &item.QueuedAt, &item.SyncedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
