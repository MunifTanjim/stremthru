package torrent_stream

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
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

func GetFilesByHashes(hashes []string) (map[string]Files, error) {
	byHash := map[string]Files{}

	if len(hashes) == 0 {
		return byHash, nil
	}

	args := make([]any, len(hashes))
	hashPlaceholders := make([]string, len(hashes))
	for i, hash := range hashes {
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
var record_streams_query_on_conflict = fmt.Sprintf(
	" ON CONFLICT (%s,%s) DO UPDATE SET %s, %s, %s, %s, %s, %s, %s, %s",
	Column.Hash,
	Column.Path,
	fmt.Sprintf(
		"%s = CASE WHEN EXCLUDED.%s IN ('dht','tor') OR ts.%s = -1 OR ts.%s IN ('','mfn') THEN EXCLUDED.%s ELSE ts.%s END",
		Column.Idx, Column.Source, Column.Idx, Column.Source, Column.Idx, Column.Idx,
	),
	fmt.Sprintf(
		"%s = CASE WHEN EXCLUDED.%s IN ('dht','tor') OR ts.%s = -1 OR ts.%s IN ('','mfn') THEN EXCLUDED.%s ELSE ts.%s END",
		Column.Size, Column.Source, Column.Size, Column.Source, Column.Size, Column.Size,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ts.%s IN ('', '*') THEN EXCLUDED.%s ELSE ts.%s END",
		Column.SId, Column.SId, Column.SId, Column.SId,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ts.%s = '' THEN EXCLUDED.%s ELSE ts.%s END",
		Column.ASId, Column.ASId, Column.ASId, Column.ASId,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ts.%s = '' THEN EXCLUDED.%s ELSE ts.%s END",
		Column.VideoHash, Column.VideoHash, Column.VideoHash, Column.VideoHash,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ts.%s IS NULL THEN EXCLUDED.%s ELSE ts.%s END",
		Column.MediaInfo, Column.MediaInfo, Column.MediaInfo, Column.MediaInfo,
	),
	fmt.Sprintf(
		"%s = CASE WHEN (EXCLUDED.%s NOT IN ('dht','tor') AND ts.%s IN ('dht','tor')) OR (EXCLUDED.%s = 'mfn' AND ts.%s != 'mfn') OR EXCLUDED.%s = '' THEN ts.%s ELSE EXCLUDED.%s END",
		Column.Source, Column.Source, Column.Source, Column.Source, Column.Source, Column.Source, Column.Source, Column.Source,
	),
	fmt.Sprintf(
		"%s = %s",
		Column.UAt, db.CurrentTimestamp,
	),
)

var recordSkipCount atomic.Int64
var recordAllowCount atomic.Int64

func GetRecordCacheStats() (skipped int64, allowed int64) {
	return recordSkipCount.Load(), recordAllowCount.Load()
}

var prevRecordSourceCache = cache.NewLRUCache[string](&cache.CacheConfig{
	Name:     "torrent_stream:prev_record_src",
	Lifetime: 1 * time.Hour,
	MaxSize:  200_000,
})

func Record(items []InsertData, discardIdx bool) error {
	if len(items) == 0 {
		return nil
	}

	errs := []error{}
	for cItems := range slices.Chunk(items, 150) {
		seenFileMap := map[string]struct{}{}
		recordSrcByKey := map[string]string{}

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
				var prevRecordSource string
				if prevRecordSourceCache.Get(key, &prevRecordSource) && (prevRecordSource == item.Source || prevRecordSource == "dht" || prevRecordSource == "tor") {
					recordSkipCount.Add(1)
					count--
					continue
				}
				recordAllowCount.Add(1)
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
				recordSrcByKey[key] = item.Source
			} else {
				log.Debug("skipped duplicate file", "hash", item.Hash, "path", item.Path)
				count--
			}
		}
		if count == 0 {
			continue
		}
		query := record_streams_query_before_values +
			util.RepeatJoin(record_streams_query_values_placeholder, count, ",") +
			record_streams_query_on_conflict
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed partially to record", "error", err)
			errs = append(errs, err)
		} else {
			log.Debug("recorded torrent stream", "count", count)
			for key, source := range recordSrcByKey {
				prevRecordSourceCache.Add(key, source)
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
