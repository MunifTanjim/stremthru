package torrent_stream

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/zeebo/xxh3"
)

func JSONBMediaInfo(mi *media_info.MediaInfo) db.JSONB[media_info.MediaInfo] {
	if mi == nil {
		return db.JSONB[media_info.MediaInfo]{Null: true}
	}
	return db.JSONB[media_info.MediaInfo]{Data: *mi}
}

type File struct {
	Path      string                `json:"p"`
	Idx       int                   `json:"i"`
	Size      int64                 `json:"s"`
	Name      string                `json:"n"`
	SId       string                `json:"sid,omitempty"`
	ASId      string                `json:"asid,omitempty"`
	Source    string                `json:"src,omitempty"`
	VideoHash string                `json:"vhash,omitempty"`
	MediaInfo *media_info.MediaInfo `json:"mi,omitempty"`

	is_video *bool `json:"-"`
}

func (f File) IsVideo() bool {
	if f.is_video != nil {
		return *f.is_video
	}
	isVideo := core.HasVideoExtension(f.Name)
	f.is_video = &isVideo
	return isVideo
}

func (f *File) Normalize() {
	if f.Name == "" && f.Path != "" {
		f.Name = filepath.Base(f.Path)
	}
}

type Files []File

func (files Files) Normalize() {
	for i := range files {
		f := &files[i]
		f.Normalize()
	}
}

func (files Files) HasVideo() bool {
	for i := range files {
		if core.HasVideoExtension(files[i].Path) {
			return true
		}
	}
	return false
}

func (files Files) Value() (driver.Value, error) {
	return json.Marshal(files)
}

func (files *Files) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to convert value to []byte")
	}
	if err := json.Unmarshal(bytes, files); err != nil {
		return err
	}
	files.Normalize()
	if len(*files) == 1 && (*files)[0].Path == "" {
		*files = (*files)[:0]
	}
	return nil
}

func _hasActualPath(f store.MagnetFile) bool {
	return strings.HasPrefix(f.Path, "/")
}

func (arr Files) ToStoreMagnetFiles(hash string) []store.MagnetFile {
	files := make([]store.MagnetFile, len(arr))
	hasActualPath := false
	hasNameAsPath := false
	for i := range arr {
		f := &arr[i]
		f.Normalize()
		files[i] = store.MagnetFile{
			Idx:       f.Idx,
			Path:      f.Path,
			Name:      f.Name,
			Size:      f.Size,
			Source:    f.Source,
			VideoHash: f.VideoHash,
			MediaInfo: f.MediaInfo,
		}
		if !hasActualPath && strings.HasPrefix(f.Path, "/") {
			hasActualPath = true
		}
		if !hasNameAsPath && !strings.HasPrefix(f.Path, "/") {
			hasNameAsPath = true
		}
	}
	if hasActualPath && hasNameAsPath {
		files = util.FilterSlice(files, _hasActualPath)
		cleanupFilesWithNameAsPath(hash, arr)
	}
	return files
}

const TableName = "torrent_stream"

type TorrentStream struct {
	Hash      string       `json:"h"`
	Path      string       `json:"p"`
	Idx       int          `json:"i"`
	Size      int64        `json:"s"`
	SId       string       `json:"sid"`
	ASId      string       `json:"asid"`
	Source    string       `json:"src"`
	VideoHash string       `json:"vhash,omitempty"`
	MediaInfo string       `json:"mi,omitempty"`
	CAt       db.Timestamp `json:"cat"`
	UAt       db.Timestamp `json:"uat"`
}

var Column = struct {
	Hash      string
	Path      string
	Idx       string
	Size      string
	SId       string
	ASId      string
	Source    string
	VideoHash string
	MediaInfo string
	CAt       string
	UAt       string
}{
	Hash:      "h",
	Path:      "p",
	Idx:       "i",
	Size:      "s",
	SId:       "sid",
	ASId:      "asid",
	Source:    "src",
	VideoHash: "vhash",
	MediaInfo: "mi",
	CAt:       "cat",
	UAt:       "uat",
}

var Columns = []string{
	Column.Hash,
	Column.Path,
	Column.Idx,
	Column.Size,
	Column.SId,
	Column.ASId,
	Column.Source,
	Column.VideoHash,
	Column.MediaInfo,
	Column.CAt,
	Column.UAt,
}

var query_cleanup_files_with_name_as_path = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = (SELECT %s FROM %s WHERE %s = ? AND %s LIKE '%s' LIMIT 1) AND %s NOT LIKE '%s'`,
	TableName,
	Column.Hash,
	Column.Hash,
	TableName,
	Column.Hash,
	Column.Path,
	"/%",
	Column.Path,
	"/%",
)

func cleanupFilesWithNameAsPath(hash string, files Files) {
	for i := range files {
		f := &files[i]
		if strings.HasPrefix(f.Path, "/") || f.Path == "" {
			continue
		}

		if f.SId != "" && f.SId != "*" {
			if _, err := db.Exec(
				fmt.Sprintf(
					`UPDATE %s SET %s = ? WHERE %s = ? AND %s LIKE '%%/%s' AND %s IN ('','*')`,
					TableName,
					Column.SId,
					Column.Hash,
					Column.Path,
					strings.ReplaceAll(f.Path, "'", "''"),
					Column.SId,
				),
				f.SId,
				hash,
			); err != nil {
				log.Error("failed to cleanup files with name as path (migrate sid)", "error", err, "hash", hash, "fpath", f.Path, "sid", f.SId)
				return
			}
		}

		if f.ASId != "" {
			if _, err := db.Exec(
				fmt.Sprintf(
					`UPDATE %s SET %s = ? WHERE %s = ? AND %s LIKE '%%/%s' AND %s = ''`,
					TableName,
					Column.ASId,
					Column.Hash,
					Column.Path,
					strings.ReplaceAll(f.Path, "'", "''"),
					Column.ASId,
				),
				f.ASId,
				hash,
			); err != nil {
				log.Error("failed to cleanup files with name as path (migrate asid)", "error", err, "hash", hash, "fpath", f.Path, "asid", f.ASId)
				return
			}
		}
	}
	_, err := db.Exec(query_cleanup_files_with_name_as_path, hash)
	if err != nil {
		log.Error("failed to cleanup files with name as path", "error", err, "hash", hash)
	} else {
		log.Debug("cleaned up files with name as path", "hash", hash)
		filesByHashCache.Remove(hash)
	}
}

var query_get_anime_file_for_kitsu = fmt.Sprintf(
	`SELECT %s, %s, %s FROM %s WHERE %s = ? AND %s = CONCAT((SELECT %s FROM %s WHERE %s = ?), ':', CAST(? AS varchar))`,
	Column.Path, Column.Idx, Column.Size,
	TableName,
	Column.Hash,
	Column.ASId,
	anime.IdMapColumn.AniDB,
	anime.IdMapTableName,
	anime.IdMapColumn.Kitsu,
)

func getAnimeFileForKitsu(hash string, asid string) (*File, error) {
	kitsuId, episode, _ := strings.Cut(strings.TrimPrefix(asid, "kitsu:"), ":")
	row := db.QueryRow(query_get_anime_file_for_kitsu, hash, kitsuId, episode)
	var file File
	if err := row.Scan(&file.Path, &file.Idx, &file.Size); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	file.Normalize()
	return &file, nil
}

var query_get_anime_file_for_mal = fmt.Sprintf(
	`SELECT %s, %s, %s FROM %s WHERE %s = ? AND %s = CONCAT((SELECT %s FROM %s WHERE %s = ?), ':', CAST(? AS varchar))`,
	Column.Path, Column.Idx, Column.Size,
	TableName,
	Column.Hash,
	Column.ASId,
	anime.IdMapColumn.AniDB,
	anime.IdMapTableName,
	anime.IdMapColumn.MAL,
)

func getAnimeFileForMAL(hash string, asid string) (*File, error) {
	malId, episode, _ := strings.Cut(strings.TrimPrefix(asid, "mal:"), ":")
	row := db.QueryRow(query_get_anime_file_for_mal, hash, malId, episode)
	var file File
	if err := row.Scan(&file.Path, &file.Idx, &file.Size); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	file.Normalize()
	return &file, nil
}

var query_get_file = fmt.Sprintf(
	"SELECT %s, %s, %s FROM %s WHERE %s = ? AND %s = ?",
	Column.Path, Column.Idx, Column.Size,
	TableName,
	Column.Hash,
	Column.SId,
)

func GetFile(hash string, sid string) (*File, error) {
	if strings.HasPrefix(sid, "kitsu:") {
		return getAnimeFileForKitsu(hash, sid)
	}
	if strings.HasPrefix(sid, "mal:") {
		return getAnimeFileForMAL(hash, sid)
	}
	row := db.QueryRow(query_get_file, hash, sid)
	var file File
	if err := row.Scan(&file.Path, &file.Idx, &file.Size); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	file.Normalize()
	return &file, nil
}

var filesByHashCache = cache.NewCache[Files](&cache.CacheConfig{
	Name:     "torrent_stream:files_by_hash",
	Lifetime: 2 * time.Hour,
	MaxSize:  400_000,
})

var readCacheHitCount atomic.Int64
var readCacheMissCount atomic.Int64

func GetReadCacheStats() (hit int64, miss int64) {
	return readCacheHitCount.Load(), readCacheMissCount.Load()
}

func GetFilesByHashes(hashes []string) (map[string]Files, error) {
	byHash := map[string]Files{}

	if len(hashes) == 0 {
		return byHash, nil
	}

	var missedHashes []string
	for _, hash := range hashes {
		var cached Files
		if filesByHashCache.Get(hash, &cached) {
			readCacheHitCount.Add(1)
			byHash[hash] = cached
		} else {
			missedHashes = append(missedHashes, hash)
		}
	}

	if len(missedHashes) == 0 {
		return byHash, nil
	}

	readCacheMissCount.Add(int64(len(missedHashes)))

	args := make([]any, len(missedHashes))
	hashPlaceholders := make([]string, len(missedHashes))
	for i, hash := range missedHashes {
		args[i] = hash
		hashPlaceholders[i] = "?"
	}

	rows, err := db.Query("SELECT h, "+db.FnJSONGroupArray+"("+db.FnJSONObject+"('i', i, 'p', p, 's', s, 'sid', sid, 'asid', asid, 'src', src, 'vhash', vhash, 'mi', jsonb(mi))) AS files FROM "+TableName+" WHERE h IN ("+strings.Join(hashPlaceholders, ",")+") GROUP BY h", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		hash := ""
		files := Files{}
		if err := rows.Scan(&hash, &files); err != nil {
			return nil, err
		}
		filesByHashCache.Add(hash, files)
		byHash[hash] = files
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return byHash, nil
}

func TrackFiles(storeCode store.StoreCode, filesByHash map[string]Files) {
	items := []InsertData{}
	for hash, files := range filesByHash {
		shouldIgnoreFiles := storeCode == store.StoreCodePremiumize && !files.HasVideo()
		if shouldIgnoreFiles {
			continue
		}
		for _, file := range files {
			if !strings.HasPrefix(file.Path, "/") {
				continue
			}
			items = append(items, InsertData{Hash: hash, File: file})
		}
	}
	discardIdx := storeCode != store.StoreCodeRealDebrid
	Record(items, discardIdx)
}

type InsertData struct {
	Hash string
	File
}

var record_streams_query_before_values = fmt.Sprintf(
	"INSERT INTO %s AS ts (%s) VALUES ",
	TableName,
	db.JoinColumnNames(
		Column.Hash,
		Column.Path,
		Column.Idx,
		Column.Size,
		Column.SId,
		Column.ASId,
		Column.Source,
		Column.VideoHash,
		Column.MediaInfo,
	),
)
var record_streams_query_values_placeholder = fmt.Sprintf("(%s)", util.RepeatJoin("?", 9, ","))

var query_record_cond_new_source_is_dht_or_tor = fmt.Sprintf(
	`EXCLUDED.%s IN ('dht','tor')`, Column.Source,
)
var query_record_cond_old_source_is_not_dht_or_tor = fmt.Sprintf(
	`ts.%s NOT IN ('dht','tor')`, Column.Source,
)

var query_record_cond_should_update_idx = fmt.Sprintf(
	`(%s AND EXCLDUED.%s != -1 AND ts.%s != EXCLUDED.%s)`,
	query_record_cond_old_source_is_not_dht_or_tor,
	Column.Idx, Column.Idx, Column.Idx,
)
var query_record_cond_should_update_size = fmt.Sprintf(
	`(%s AND EXCLUDED.%s != -1 AND ts.%s != EXCLUDED.%s)`,
	query_record_cond_old_source_is_not_dht_or_tor,
	Column.Size, Column.Size, Column.Size,
)
var query_record_cond_should_update_sid = fmt.Sprintf(
	`(EXCLUDED.%s NOT IN ('', '*') AND ts.%s != EXCLUDED.%s)`,
	Column.SId, Column.SId, Column.SId,
)
var query_record_cond_should_update_asid = fmt.Sprintf(
	`(EXCLUDED.%s != '' AND ts.%s != EXCLUDED.%s)`,
	Column.ASId, Column.ASId, Column.ASId,
)
var query_record_cond_should_update_vhash = fmt.Sprintf(
	`(EXCLUDED.%s != '' AND ts.%s = '')`,
	Column.VideoHash, Column.VideoHash,
)
var query_record_cond_should_update_mi = fmt.Sprintf(
	`(EXCLUDED.%s IS NOT NULL AND ts.%s IS NULL)`,
	Column.MediaInfo, Column.MediaInfo,
)
var query_record_cond_should_update_src = fmt.Sprintf(
	`((%s OR %s) AND (EXCLUDED.%s != 'mfn' OR ts.%s = 'mfn') AND EXCLUDED.%s != '')`,
	query_record_cond_new_source_is_dht_or_tor, query_record_cond_old_source_is_not_dht_or_tor,
	Column.Source, Column.Source,
	Column.Source,
)

var query_record_on_conflict_set_cond = func(col, cond string) string {
	return fmt.Sprintf(
		"%s = CASE WHEN %s THEN EXCLUDED.%s ELSE ts.%s END",
		col, cond, col, col,
	)
}

var record_streams_query_on_conflict = fmt.Sprintf(
	" ON CONFLICT (%s,%s) DO UPDATE SET %s, %s, %s, %s, %s, %s, %s, %s WHERE %s",
	Column.Hash,
	Column.Path,
	query_record_on_conflict_set_cond(Column.Idx, query_record_cond_should_update_idx),
	query_record_on_conflict_set_cond(Column.Size, query_record_cond_should_update_size),
	query_record_on_conflict_set_cond(Column.SId, query_record_cond_should_update_sid),
	query_record_on_conflict_set_cond(Column.ASId, query_record_cond_should_update_asid),
	query_record_on_conflict_set_cond(Column.VideoHash, query_record_cond_should_update_vhash),
	query_record_on_conflict_set_cond(Column.MediaInfo, query_record_cond_should_update_mi),
	query_record_on_conflict_set_cond(Column.Source, query_record_cond_should_update_src),
	fmt.Sprintf("%s = %s", Column.UAt, db.CurrentTimestamp),
	strings.Join([]string{
		query_record_cond_should_update_idx,
		query_record_cond_should_update_size,
		query_record_cond_should_update_sid,
		query_record_cond_should_update_asid,
		query_record_cond_should_update_vhash,
		query_record_cond_should_update_mi,
		query_record_cond_should_update_src,
	}, " OR "),
)

func get_record_query(count int) string {
	return record_streams_query_before_values +
		util.RepeatJoin(record_streams_query_values_placeholder, count, ",") +
		record_streams_query_on_conflict
}

type prevRecordData struct {
	Source      string
	Fingerprint uint64
}

var prevRecordCache = cache.NewLRUCache[prevRecordData](&cache.CacheConfig{
	Name:     "torrent_stream:prev_record",
	Lifetime: 4 * time.Hour,
	MaxSize:  1_000_000,
})

var writeCacheHitCount atomic.Int64
var writeCacheMissCount atomic.Int64

func GetWriteCacheStats() (hit int64, miss int64) {
	return writeCacheHitCount.Load(), writeCacheMissCount.Load()
}

func Record(items []InsertData, discardIdx bool) error {
	if len(items) == 0 {
		return nil
	}

	errs := []error{}
	for cItems := range slices.Chunk(items, 150) {
		seenFileMap := map[string]struct{}{}
		recordDataByKey := map[string]prevRecordData{}

		count := len(cItems)
		args := make([]any, 0, count*9)
		for i := range cItems {
			item := &cItems[i]
			if !strings.HasPrefix(item.Path, "/") {
				continue
			}

			idx := item.Idx
			if discardIdx && item.Source != "dht" && item.Source != "tor" {
				idx = -1
			}
			sid := item.SId
			if sid == "" {
				sid = "*"
			}
			key := item.Hash + ":" + item.Path
			if _, seen := seenFileMap[key]; !seen {
				seenFileMap[key] = struct{}{}
				var fpBuf bytes.Buffer
				fpBuf.WriteString(strconv.Itoa(idx))
				fpBuf.WriteString(strconv.FormatInt(item.Size, 10))
				fpBuf.WriteString(sid)
				fpBuf.WriteString(item.ASId)
				fpBuf.WriteString(item.VideoHash)
				fpBuf.WriteString(strconv.FormatBool(item.MediaInfo != nil))
				fingerprint := xxh3.Hash(fpBuf.Bytes())
				var prev prevRecordData
				if prevRecordCache.Get(key, &prev) && (prev.Source == item.Source || prev.Source == "dht" || prev.Source == "tor" || prev.Fingerprint == fingerprint) {
					writeCacheHitCount.Add(1)
					count--
					continue
				}
				writeCacheMissCount.Add(1)
				args = append(args,
					item.Hash,
					item.Path,
					idx,
					item.Size,
					sid,
					item.ASId,
					item.Source,
					item.VideoHash,
					JSONBMediaInfo(item.MediaInfo),
				)
				recordDataByKey[key] = prevRecordData{Source: item.Source, Fingerprint: fingerprint}
			} else {
				log.Debug("skipped duplicate file", "hash", item.Hash, "path", item.Path)
				count--
			}
		}
		if count == 0 {
			continue
		}
		_, err := db.Exec(get_record_query(count), args...)
		if err != nil {
			log.Error("failed partially to record", "error", err)
			errs = append(errs, err)
		} else {
			log.Debug("recorded torrent stream", "count", count)
			invalidatedHashes := util.NewSet[string]()
			for key, data := range recordDataByKey {
				prevRecordCache.Add(key, data)
				hash, _, _ := strings.Cut(key, ":")
				if !invalidatedHashes.Has(hash) {
					filesByHashCache.Remove(hash)
					invalidatedHashes.Add(hash)
				}
			}
		}
	}

	return errors.Join(errs...)
}

var query_has_media_info = fmt.Sprintf(
	"SELECT 1 FROM %s WHERE %s = ? AND %s = ? AND %s IS NOT NULL LIMIT 1",
	TableName,
	Column.Hash,
	Column.Path,
	Column.MediaInfo,
)

func HasMediaInfo(hash, path string) bool {
	row := db.QueryRow(query_has_media_info, hash, path)
	err := row.Scan(new(int))
	return err == nil
}

var query_set_media_info = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ?",
	TableName,
	Column.MediaInfo,
	Column.UAt, db.CurrentTimestamp,
	Column.Hash,
	Column.Path,
)

func SetMediaInfo(hash, path string, mediaInfo *media_info.MediaInfo) error {
	_, err := db.Exec(query_set_media_info, JSONBMediaInfo(mediaInfo), hash, path)
	if err == nil {
		filesByHashCache.Remove(hash)
	}
	return err
}

var tag_strem_id_query = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = ? WHERE %s = ? AND %s = ? AND %s IN ('', '*')",
	TableName,
	Column.SId,
	Column.UAt,
	Column.Hash,
	Column.Path,
	Column.SId,
)

func TagStremId(hash string, filepath string, sid string) {
	if filepath == "" {
		return
	}
	if !strings.HasPrefix(sid, "tt") {
		return
	}
	_, err := db.Exec(tag_strem_id_query, sid, db.Timestamp{Time: time.Now()}, hash, filepath)
	if err != nil {
		log.Error("failed to tag strem id", "error", err, "hash", hash, "fpath", filepath, "sid", sid)
	} else {
		log.Debug("tagged strem id", "hash", hash, "fpath", filepath, "sid", sid)
		filesByHashCache.Remove(hash)
	}
}

var query_tag_anime_strem_id = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ? AND %s = ''`,
	TableName,
	Column.ASId,
	Column.UAt,
	db.CurrentTimestamp,
	Column.Hash,
	Column.Path,
	Column.ASId,
)

func TagAnimeStremId(hash string, filepath string, sid string) {
	if filepath == "" {
		return
	}
	var anidbId, episode string
	var err error
	if kitsuSid, ok := strings.CutPrefix(sid, "kitsu:"); ok {
		kitsuId, kitsuEpisode, _ := strings.Cut(kitsuSid, ":")
		anidbId, _, err = anime.GetAniDBIdByKitsuId(kitsuId)
		episode = kitsuEpisode
	} else if malSid, ok := strings.CutPrefix(sid, "mal:"); ok {
		malId, malEpisode, _ := strings.Cut(malSid, ":")
		anidbId, _, err = anime.GetAniDBIdByMALId(malId)
		episode = malEpisode
	} else {
		return
	}
	if err != nil {
		log.Error("failed to get anidb id for anime", "error", err, "sid", sid)
		return
	}
	asid := anidbId + ":" + episode
	_, err = db.Exec(query_tag_anime_strem_id, asid, hash, filepath)
	if err != nil {
		log.Error("failed to tag anime strem id", "error", err, "hash", hash, "fpath", filepath, "asid", asid, "strem_id", sid)
	} else {
		log.Debug("tagged anime strem id", "hash", hash, "fpath", filepath, "asid", asid, "strem_id", sid)
		filesByHashCache.Remove(hash)
	}
}

func GetStremIdByHashes(hashes []string) (*url.Values, error) {
	byHash := &url.Values{}
	count := len(hashes)
	if count == 0 {
		return byHash, nil
	}

	query := fmt.Sprintf(
		`SELECT %s, %s FROM %s WHERE %s IN (%s) AND %s like 'tt%%' GROUP BY %s, %s`,
		Column.Hash, Column.SId,
		TableName,
		Column.Hash, util.RepeatJoin("?", count, ","),
		Column.SId,
		Column.Hash,
		Column.SId,
	)
	args := make([]any, count)
	for i, hash := range hashes {
		args[i] = hash
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return byHash, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash, sid string
		if err := rows.Scan(&hash, &sid); err != nil {
			return byHash, err
		}
		byHash.Add(hash, sid)
	}

	if err := rows.Err(); err != nil {
		return byHash, err
	}
	return byHash, nil
}

type Stats struct {
	TotalCount    int            `json:"total_count"`
	CountBySource map[string]int `json:"count_by_source"`
}

var stats_query = fmt.Sprintf(
	"SELECT %s, COUNT(%s) FROM %s WHERE %s NOT IN ('', '*') AND %s != '' GROUP BY %s",
	Column.Source,
	Column.Path,
	TableName,
	Column.SId,
	Column.Source,
	Column.Source,
)

func GetStats() (*Stats, error) {
	var stats Stats
	rows, err := db.Query(stats_query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.CountBySource = make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		stats.CountBySource[source] = count
		stats.TotalCount += count
	}
	return &stats, nil
}
