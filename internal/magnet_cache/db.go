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

var Column = struct {
	Store      string
	Hash       string
	IsCached   string
	ModifiedAt string
}{
	Store:      "store",
	Hash:       "hash",
	IsCached:   "is_cached",
	ModifiedAt: "modified_at",
}

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

var magnetInfoByHashCache = cache.NewCache[MagnetCache](&cache.CacheConfig{
	Name:     "magnet_cache:info_by_hash",
	Lifetime: 2 * time.Hour,
	MaxSize:  200_000,
})

var getReadCacheHitCount atomic.Int64
var getReadCacheMissCount atomic.Int64

func GetReadCacheStats() (hit int64, miss int64) {
	return getReadCacheHitCount.Load(), getReadCacheMissCount.Load()
}

func matchesSidFilter(mc *MagnetCache, sid string) bool {
	if sid == "" || !mc.IsCached {
		return true
	}
	for _, f := range mc.Files {
		if f.SId == sid || f.SId == "*" {
			return true
		}
	}
	return false
}

var query_get_by_hashes = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s IN `,
	db.JoinColumnNames(
		Column.Store,
		Column.Hash,
		Column.IsCached,
		Column.ModifiedAt,
	),
	TableName,
	Column.Store,
	Column.Hash,
)

func GetByHashes(storeCode store.StoreCode, hashes []string, sid string) ([]MagnetCache, error) {
	if len(hashes) == 0 {
		return []MagnetCache{}, nil
	}

	filesByHash, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		return nil, err
	}

	mcs := []MagnetCache{}
	var missedHashes []string

	for _, hash := range hashes {
		cacheKey := writeCacheKey(storeCode, hash)
		var cached MagnetCache
		if magnetInfoByHashCache.Get(cacheKey, &cached) {
			getReadCacheHitCount.Add(1)
			if files, ok := filesByHash[cached.Hash]; ok && len(files) > 0 {
				cached.Files = files
			}
			if matchesSidFilter(&cached, sid) {
				mcs = append(mcs, cached)
			}
		} else {
			missedHashes = append(missedHashes, hash)
		}
	}

	getReadCacheMissCount.Add(int64(len(missedHashes)))

	if len(missedHashes) == 0 {
		return mcs, nil
	}

	args_len := len(missedHashes) + 1
	arg_idx := 0
	args := make([]any, args_len)

	query := query_get_by_hashes

	args[arg_idx] = storeCode
	arg_idx += 1
	hashPlaceholders := make([]string, len(missedHashes))
	for i, hash := range missedHashes {
		hashPlaceholders[i] = "?"
		args[arg_idx+i] = hash
	}

	query += "(" + strings.Join(hashPlaceholders, ",") + ")"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		smc := MagnetCache{}
		if err := rows.Scan(&smc.Store, &smc.Hash, &smc.IsCached, &smc.ModifiedAt); err != nil {
			return nil, err
		}

		cacheKey := writeCacheKey(storeCode, smc.Hash)
		magnetInfoByHashCache.Add(cacheKey, smc)

		if files, ok := filesByHash[smc.Hash]; ok && len(files) > 0 {
			smc.Files = files
		}

		if matchesSidFilter(&smc, sid) {
			mcs = append(mcs, smc)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mcs, nil
}

var prevIsCachedCache = cache.NewLRUCache[bool](&cache.CacheConfig{
	Name:     "magnet_cache:prev_is_cached",
	Lifetime: 4 * time.Hour,
	MaxSize:  200_000,
})

var writeCacheHitCount atomic.Int64
var writeCacheMissCount atomic.Int64

func GetWriteCacheStats() (hit int64, miss int64) {
	return writeCacheHitCount.Load(), writeCacheMissCount.Load()
}

func writeCacheKey(storeCode store.StoreCode, hash string) string {
	return string(storeCode) + ":" + hash
}

func Touch(storeCode store.StoreCode, hash string, files torrent_stream.Files, isCached bool, skipFileTracking bool) {
	cacheKey := writeCacheKey(storeCode, hash)
	var prevIsCached bool
	if prevIsCachedCache.Get(cacheKey, &prevIsCached) && prevIsCached == isCached {
		writeCacheHitCount.Add(1)
	} else {
		writeCacheMissCount.Add(1)
		buf := bytes.NewBuffer([]byte("INSERT INTO " + TableName))
		var result sql.Result
		var err error
		is_cached := db.BooleanFalse
		if isCached {
			is_cached = db.BooleanTrue
		}
		buf.WriteString(" (store, hash, is_cached) VALUES (?, ?, " + is_cached + ") ON CONFLICT (store, hash) DO UPDATE SET is_cached = EXCLUDED.is_cached, modified_at = " + db.CurrentTimestamp + " WHERE " + TableName + ".is_cached != EXCLUDED.is_cached OR " + TableName + ".modified_at < ?")
		result, err = db.Exec(buf.String(), storeCode, hash, db.Timestamp{Time: time.Now().Add(-24 * time.Hour)})
		if err == nil {
			_, err = result.RowsAffected()
		}
		if err != nil {
			mcLog.Error("failed to touch", "error", err)
			return
		}
		prevIsCachedCache.Add(cacheKey, isCached)
		magnetInfoByHashCache.Remove(cacheKey)
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
	` ON CONFLICT (store, hash) DO UPDATE SET is_cached = EXCLUDED.is_cached, modified_at = %s WHERE %s.is_cached != EXCLUDED.is_cached OR %s.modified_at < ?`,
	db.CurrentTimestamp,
	TableName,
	TableName,
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
		cacheKey := writeCacheKey(storeCode, hash)
		var prevIsCached bool
		if prevIsCachedCache.Get(cacheKey, &prevIsCached) && prevIsCached == is_cached {
			writeCacheHitCount.Add(1)
			continue
		}

		writeCacheMissCount.Add(1)
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

	staleThreshold := db.Timestamp{Time: time.Now().Add(-24 * time.Hour)}

	if hit_count > 0 {
		hit_query.WriteString(query_bulk_touch_on_conflict)
		hit_args = append(hit_args, staleThreshold)
		_, err := db.Exec(hit_query.String(), hit_args...)
		if err != nil {
			mcLog.Error("failed to touch hits", "error", err)
		} else {
			for _, key := range hitCacheKeys {
				prevIsCachedCache.Add(key, true)
				magnetInfoByHashCache.Remove(key)
			}
		}
	}

	if !skipFileTracking && len(filesByHash) > 0 {
		torrent_stream.TrackFiles(storeCode, filesByHash)
	}

	if miss_count > 0 {
		miss_query.WriteString(query_bulk_touch_on_conflict)
		miss_args = append(miss_args, staleThreshold)
		_, err := db.Exec(miss_query.String(), miss_args...)
		if err != nil {
			mcLog.Error("failed to touch misses", "error", err)
		} else {
			for _, key := range missCacheKeys {
				prevIsCachedCache.Add(key, false)
				magnetInfoByHashCache.Remove(key)
			}
		}
	}
}
