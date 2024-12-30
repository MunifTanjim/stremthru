package magnet_cache

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/store"
)

const TableName = "magnet_cache"

type File struct {
	Idx  int    `json:"i"`
	Name string `json:"n"`
	Size int    `json:"s"`
	SId  string `json:"-"`
}

type Files []File

func (files Files) Value() (driver.Value, error) {
	return json.Marshal(files)
}

func (files *Files) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, files)
}

func (arr Files) ToStoreMagnetFile() []store.MagnetFile {
	files := make([]store.MagnetFile, len(arr))
	for i, f := range arr {
		files[i] = store.MagnetFile{
			Idx:  f.Idx,
			Name: f.Name,
			Size: f.Size,
		}
	}
	return files
}

type MagnetCache struct {
	Store      store.StoreCode
	Hash       string
	IsCached   bool
	ModifiedAt db.Timestamp
	Files      Files
}

// If Buddy is available, refresh data more frequently.
var cachedStaleTime = func() time.Duration {
	if config.HasBuddy {
		return 1 * time.Hour
	}
	return 12 * time.Hour
}()
var uncachedStaleTime = func() time.Duration {
	if config.HasBuddy {
		return 30 * time.Minute
	}
	return 1 * time.Hour
}()

func (mc MagnetCache) IsStale() bool {
	if mc.IsCached {
		return mc.ModifiedAt.Before(time.Now().Add(-cachedStaleTime))
	}
	return mc.ModifiedAt.Before(time.Now().Add(-uncachedStaleTime))
}

func GetByHashes(store store.StoreCode, hashes []string, sid string) ([]MagnetCache, error) {
	filesByHash, err := GetFilesByHashes(hashes)
	if err != nil {
		return nil, err
	}

	args_len := len(hashes) + 1
	if sid != "" {
		args_len += 1
	}
	arg_idx := 0
	args := make([]interface{}, args_len)

	query := "SELECT store, hash, is_cached, modified_at, files FROM " + TableName
	if sid != "" {
		query += " LEFT JOIN " + FileTableName + " ON " + TableName + ".hash = " + FileTableName + ".h WHERE (is_cached = " + db.BooleanFalse + " OR " + FileTableName + ".sid IN (?, '*')) AND"
		args[arg_idx] = sid
		arg_idx += 1
	} else {
		query += " WHERE"
	}

	args[arg_idx] = store
	arg_idx += 1
	hashPlaceholders := make([]string, len(hashes))
	for i, hash := range hashes {
		hashPlaceholders[i] = "?"
		args[arg_idx+i] = hash
	}

	query += " store = ? AND hash IN (" + strings.Join(hashPlaceholders, ",") + ")"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mcs := []MagnetCache{}
	for rows.Next() {
		smc := MagnetCache{}
		if err := rows.Scan(&smc.Store, &smc.Hash, &smc.IsCached, &smc.ModifiedAt, &smc.Files); err != nil {
			return nil, err
		}
		if files, ok := filesByHash[smc.Hash]; ok && len(files) > 0 {
			smc.Files = files
		}
		mcs = append(mcs, smc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mcs, nil
}

func Touch(store store.StoreCode, hash string, files Files, sid string) {
	buf := bytes.NewBuffer([]byte("INSERT INTO " + TableName))
	var result sql.Result
	var err error
	if len(files) == 0 {
		buf.WriteString(" (store, hash, is_cached) VALUES (?, ?, false) ON CONFLICT (store, hash) DO UPDATE SET is_cached = excluded.is_cached, modified_at = " + db.CurrentTimestamp)
		result, err = db.Exec(buf.String(), store, hash)
	} else {
		buf.WriteString(" (store, hash, is_cached, files) VALUES (?, ?, true, ?) ON CONFLICT (store, hash) DO UPDATE SET is_cached = excluded.is_cached, files = excluded.files, modified_at = " + db.CurrentTimestamp)
		result, err = db.Exec(buf.String(), store, hash, files)
	}
	if err == nil {
		_, err = result.RowsAffected()
	}
	if err != nil {
		log.Printf("[magnet_cache] failed to touch: %v\n", err)
		return
	}
	TrackFiles(hash, files, sid)
}

func BulkTouch(store store.StoreCode, filesByHash map[string]Files, sid string) {
	var hit_query strings.Builder
	hit_query.WriteString("INSERT INTO " + TableName + " (store,hash,is_cached,files) VALUES ")
	hit_placeholder := "(?,?,true,?)"
	hit_count := 0

	var miss_query strings.Builder
	miss_query.WriteString("INSERT INTO " + TableName + " (store,hash,is_cached) VALUES ")
	miss_placeholder := "(?,?,false)"
	miss_count := 0

	var hit_args []interface{}
	var miss_args []interface{}

	for hash, files := range filesByHash {
		if len(files) == 0 {
			if miss_count > 0 {
				miss_query.WriteString(",")
			}
			miss_query.WriteString(miss_placeholder)
			miss_args = append(miss_args, store, hash)
			miss_count++
		} else {
			if hit_count > 0 {
				hit_query.WriteString(",")
			}
			hit_query.WriteString(hit_placeholder)
			hit_args = append(hit_args, store, hash, files)
			hit_count++
		}
	}

	if hit_count > 0 {
		hit_query.WriteString(" ON CONFLICT (store, hash) DO UPDATE SET is_cached = excluded.is_cached, files = excluded.files, modified_at = " + db.CurrentTimestamp)
		_, err := db.Exec(hit_query.String(), hit_args...)
		if err != nil {
			log.Printf("[magnet_cache] failed to touch hits: %v\n", err)
		}
		BulkTrackFiles(filesByHash, sid)
	}

	if miss_count > 0 {
		miss_query.WriteString(" ON CONFLICT (store, hash) DO UPDATE SET is_cached = excluded.is_cached, modified_at = " + db.CurrentTimestamp)
		_, err := db.Exec(miss_query.String(), miss_args...)
		if err != nil {
			log.Printf("[magnet_cache] failed to touch misses: %v\n", err)
		}
	}
}
