package magnet_cache

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/store"
)

const TableName = "magnet_cache"

var mcLog = logger.Scoped(TableName)

type MagnetCache struct {
	Store      store.StoreCode
	Hash       string
	IsCached   bool
	ModifiedAt db.Timestamp
	Files      torrent_stream.Files
}

func (mc MagnetCache) IsStale() bool {
	staleTime := config.StoreContentCachedStaleTime.GetStaleTime(mc.IsCached, string(mc.Store.Name()))
	if config.HasBuddy {
		// If Buddy is available, refresh data more frequently.
		staleTime = staleTime / 2
	}
	return mc.ModifiedAt.Before(time.Now().Add(-staleTime))
}

type magnetCacheRow struct {
	Store      store.StoreCode `json:"s"`
	Hash       string          `json:"h"`
	IsCached   bool            `json:"c"`
	ModifiedAt db.Timestamp    `json:"m"`
}

var magnetCacheRowCache = cache.NewCache[magnetCacheRow](&cache.CacheConfig{
	Name:     "magnet_cache:row",
	Lifetime: 2 * time.Minute,
})

func magnetCacheKey(storeCode store.StoreCode, hash string) string {
	return string(storeCode) + ":" + hash
}

func GetByHashes(storeCode store.StoreCode, hashes []string, sid string) ([]MagnetCache, error) {
	if len(hashes) == 0 {
		return []MagnetCache{}, nil
	}

	filesByHash, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		return nil, err
	}

	// Try cache first for each hash
	uncachedHashes := make([]string, 0, len(hashes))
	cachedRows := map[string]magnetCacheRow{}
	for _, hash := range hashes {
		var row magnetCacheRow
		if magnetCacheRowCache.Get(magnetCacheKey(storeCode, hash), &row) {
			cachedRows[hash] = row
		} else {
			uncachedHashes = append(uncachedHashes, hash)
		}
	}

	// Query DB for uncached hashes
	if len(uncachedHashes) > 0 {
		args := make([]any, 1+len(uncachedHashes))
		args[0] = storeCode
		hashPlaceholders := make([]string, len(uncachedHashes))
		for i, hash := range uncachedHashes {
			hashPlaceholders[i] = "?"
			args[1+i] = hash
		}

		query := "SELECT store, hash, is_cached, modified_at FROM " + TableName + " WHERE store = ? AND hash IN (" + strings.Join(hashPlaceholders, ",") + ")"

		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var row magnetCacheRow
			if err := rows.Scan(&row.Store, &row.Hash, &row.IsCached, &row.ModifiedAt); err != nil {
				return nil, err
			}
			cachedRows[row.Hash] = row
			magnetCacheRowCache.Add(magnetCacheKey(storeCode, row.Hash), row)
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Build result, applying sid filter if needed
	mcs := []MagnetCache{}
	for _, hash := range hashes {
		row, ok := cachedRows[hash]
		if !ok {
			continue
		}

		// When sid is set, skip cached items that don't have matching files
		if sid != "" && row.IsCached {
			files, hasFiles := filesByHash[hash]
			if !hasFiles || !filesMatchSid(files, sid) {
				continue
			}
		}

		mc := MagnetCache{
			Store:      row.Store,
			Hash:       row.Hash,
			IsCached:   row.IsCached,
			ModifiedAt: row.ModifiedAt,
		}
		if files, ok := filesByHash[hash]; ok && len(files) > 0 {
			mc.Files = files
		}
		mcs = append(mcs, mc)
	}

	return mcs, nil
}

func filesMatchSid(files torrent_stream.Files, sid string) bool {
	for _, f := range files {
		if f.SId == sid || f.SId == "*" {
			return true
		}
	}
	return false
}

var touchSkipCount atomic.Int64
var touchAllowCount atomic.Int64

func GetTouchCacheStats() (skipped int64, allowed int64) {
	return touchSkipCount.Load(), touchAllowCount.Load()
}

var prevIsCachedCache = cache.NewLRUCache[bool](&cache.CacheConfig{
	Name:     "magnet_cache:prev_is_cached",
	Lifetime: 4 * time.Hour,
	MaxSize:  200_000,
})

func touchCacheKey(storeCode store.StoreCode, hash string) string {
	return string(storeCode) + ":" + hash
}

func Touch(storeCode store.StoreCode, hash string, files torrent_stream.Files, isCached bool, skipFileTracking bool) {
	cacheKey := touchCacheKey(storeCode, hash)
	var prevIsCached bool
	if prevIsCachedCache.Get(cacheKey, &prevIsCached) && prevIsCached == isCached {
		touchSkipCount.Add(1)
	} else {
		touchAllowCount.Add(1)
		buf := bytes.NewBuffer([]byte("INSERT INTO " + TableName))
		var result sql.Result
		var err error
		is_cached := db.BooleanFalse
		if isCached {
			is_cached = db.BooleanTrue
		}
		buf.WriteString(" (store, hash, is_cached) VALUES (?, ?, " + is_cached + ") ON CONFLICT (store, hash) DO UPDATE SET is_cached = EXCLUDED.is_cached, modified_at = " + db.CurrentTimestamp)
		result, err = db.Exec(buf.String(), storeCode, hash)
		if err == nil {
			_, err = result.RowsAffected()
		}
		if err != nil {
			mcLog.Error("failed to touch", "error", err)
			return
		}
		prevIsCachedCache.Add(cacheKey, isCached)
		magnetCacheRowCache.Remove(cacheKey)
	}
	if !skipFileTracking {
		torrent_stream.TrackFiles(storeCode, map[string]torrent_stream.Files{hash: files})
	}
}

var query_bulk_touch_before_values = fmt.Sprintf(
	"INSERT INTO %s (store,hash,is_cached) VALUES ",
	TableName,
)
var query_bulk_touch_on_conflict = fmt.Sprintf(
	` ON CONFLICT (store, hash) DO UPDATE SET is_cached = EXCLUDED.is_cached, modified_at = %s`,
	db.CurrentTimestamp,
)

func BulkTouch(storeCode store.StoreCode, filesByHash map[string]torrent_stream.Files, cached map[string]bool, skipFileTracking bool) {
	var hit_query strings.Builder
	hit_query.WriteString(query_bulk_touch_before_values)
	hit_placeholder := "(?,?,true)"
	hit_count := 0

	var miss_query strings.Builder
	miss_query.WriteString(query_bulk_touch_before_values)
	miss_placeholder := "(?,?,false)"
	miss_count := 0

	var hit_args []any
	var miss_args []any

	if cached == nil {
		cached = map[string]bool{}
	}

	for hash, files := range filesByHash {
		if len(files) > 0 {
			cached[hash] = true
		}
	}

	hitCacheKeys := []string{}
	missCacheKeys := []string{}

	for hash, is_cached := range cached {
		cacheKey := touchCacheKey(storeCode, hash)
		var prevIsCached bool
		if prevIsCachedCache.Get(cacheKey, &prevIsCached) && prevIsCached == is_cached {
			touchSkipCount.Add(1)
			continue
		}

		touchAllowCount.Add(1)
		if !is_cached {
			if miss_count > 0 {
				miss_query.WriteString(",")
			}
			miss_query.WriteString(miss_placeholder)
			miss_args = append(miss_args, storeCode, hash)
			miss_count++
			missCacheKeys = append(missCacheKeys, cacheKey)
		} else {
			if hit_count > 0 {
				hit_query.WriteString(",")
			}
			hit_query.WriteString(hit_placeholder)
			hit_args = append(hit_args, storeCode, hash)
			hit_count++
			hitCacheKeys = append(hitCacheKeys, cacheKey)
		}
	}

	if hit_count > 0 {
		hit_query.WriteString(query_bulk_touch_on_conflict)
		_, err := db.Exec(hit_query.String(), hit_args...)
		if err != nil {
			mcLog.Error("failed to touch hits", "error", err)
		} else {
			for _, key := range hitCacheKeys {
				prevIsCachedCache.Add(key, true)
				magnetCacheRowCache.Remove(key)
			}
		}
	}

	if !skipFileTracking && len(filesByHash) > 0 {
		torrent_stream.TrackFiles(storeCode, filesByHash)
	}

	if miss_count > 0 {
		miss_query.WriteString(query_bulk_touch_on_conflict)
		_, err := db.Exec(miss_query.String(), miss_args...)
		if err != nil {
			mcLog.Error("failed to touch misses", "error", err)
		} else {
			for _, key := range missCacheKeys {
				prevIsCachedCache.Add(key, false)
				magnetCacheRowCache.Remove(key)
			}
		}
	}
}
