package torrent_stream_syncinfo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
)

var pullCache = cache.NewLRUCache[time.Time](&cache.CacheConfig{
	Lifetime:      1 * time.Hour,
	Name:          "torrent_stream_syncinfo:pull",
	LocalCapacity: 8192,
})

var pushCache = cache.NewLRUCache[time.Time](&cache.CacheConfig{
	Lifetime:      1 * time.Hour,
	Name:          "torrent_stream_syncinfo:push",
	LocalCapacity: 8192,
})

type TorrentStreamSyncInfo struct {
	SId      string       `json:"sid"`
	PulledAt db.Timestamp `json:"pulled_at"`
	PushedAt db.Timestamp `json:"pushed_at"`
}

const TableName = "torrent_stream_syncinfo"

type ColumnStruct struct {
	SId      string
	PulledAt string
	PushedAt string
}

var Column = ColumnStruct{
	SId:      "sid",
	PulledAt: "pulled_at",
	PushedAt: "pushed_at",
}

var Columns = []string{
	Column.SId,
	Column.PulledAt,
	Column.PushedAt,
}

var staleTime = 24 * time.Hour

func ShouldPull(sid string) bool {
	sid, _, _ = strings.Cut(sid, ":")

	var syncedAt db.Timestamp
	if !pullCache.Get(sid, &syncedAt.Time) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", Column.PulledAt, TableName, Column.SId)
		row := db.QueryRow(query, sid)
		if err := row.Scan(&syncedAt); err != nil {
			if err != sql.ErrNoRows {
				log.Error("failed to get torrent stream sync info", "error", err, "sid", sid)
			}
			syncedAt.Time = time.Unix(0, 0)
		}
		if err := pullCache.Add(sid, syncedAt.Time); err != nil {
			log.Error("failed to add to pull cache", "error", err, "sid", sid)
		}
	}

	return syncedAt.Time.Before(time.Now().Add(-staleTime))
}

func ShouldPush(sid string) bool {
	sid, _, _ = strings.Cut(sid, ":")

	var syncedAt db.Timestamp
	if !pushCache.Get(sid, &syncedAt.Time) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", Column.PushedAt, TableName, Column.SId)
		row := db.QueryRow(query, sid)
		if err := row.Scan(&syncedAt); err != nil {
			if err != sql.ErrNoRows {
				log.Error("failed to get torrent stream sync info", "error", err, "sid", sid)
			}
			syncedAt.Time = time.Unix(0, 0)
		}
		if err := pushCache.Add(sid, syncedAt.Time); err != nil {
			log.Error("failed to add to push cache", "error", err, "sid", sid)
		}
	}

	return syncedAt.Time.Before(time.Now().Add(-staleTime))
}

func MarkPulled(sid string) {
	sid, _, _ = strings.Cut(sid, ":")
	query := fmt.Sprintf(
		"INSERT INTO %s (%s,%s) VALUES (?,%s) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s",
		TableName,
		Column.SId,
		Column.PulledAt,
		db.CurrentTimestamp,
		Column.SId,
		Column.PulledAt,
		Column.PulledAt,
	)
	_, err := db.Exec(query, sid)
	if err == nil {
		err = pullCache.Add(sid, time.Now())
	}
	if err != nil {
		log.Error("failed to mark pulled", "error", err, "sid", sid)
	}
}

func MarkPushed(sid string) {
	sid, _, _ = strings.Cut(sid, ":")
	query := fmt.Sprintf(
		"INSERT INTO %s (%s,%s) VALUES (?,%s) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s",
		TableName,
		Column.SId,
		Column.PushedAt,
		db.CurrentTimestamp,
		Column.SId,
		Column.PushedAt,
		Column.PushedAt,
	)
	_, err := db.Exec(query, sid)
	if err == nil {
		err = pushCache.Add(sid, time.Now())
	}
	if err != nil {
		log.Error("failed to mark pushed", "error", err, "sid", sid)
	}
}
