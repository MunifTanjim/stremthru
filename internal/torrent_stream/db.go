package torrent_stream

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

type File struct {
	Name string `json:"n"`
	Idx  int    `json:"i"`
	Size int64  `json:"s"`
	SId  string `json:"-"`
}

type Files []File

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

const TableName = "torrent_stream"

type TorrentStream struct {
	Hash   string       `json:"h"`
	Name   string       `json:"n"`
	Idx    int          `json:"i"`
	Size   int64        `json:"s"`
	SId    string       `json:"sid"`
	Source string       `json:"src"`
	CAt    db.Timestamp `json:"cat"`
	UAt    db.Timestamp `json:"uat"`
}

type ColumnStruct struct {
	Hash   string
	Name   string
	Idx    string
	Size   string
	SId    string
	Source string
	CAt    string
	UAt    string
}

var Column = ColumnStruct{
	Hash:   "h",
	Name:   "n",
	Idx:    "i",
	Size:   "s",
	SId:    "sid",
	Source: "src",
	CAt:    "cat",
	UAt:    "uat",
}

var Columns = []string{
	Column.Hash,
	Column.SId,
	Column.Name,
	Column.Idx,
	Column.Size,
	Column.Source,
	Column.CAt,
	Column.UAt,
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

	rows, err := db.Query("SELECT h, "+db.FnJSONGroupArray+"("+db.FnJSONObject+"('i', i, 'n', n, 's', s)) AS files FROM "+TableName+" WHERE h IN ("+strings.Join(hashPlaceholders, ",")+") GROUP BY h", args...)
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

func TrackFiles(hash string, files Files, discardIdx bool) {
	if len(files) == 0 {
		return
	}

	var query strings.Builder
	query.WriteString("INSERT INTO " + TableName + " (h,i,n,s,sid) VALUES ")
	placeholder := "(?,?,?,?,?)"
	count := 0

	var args []any

	for _, file := range files {
		if count > 0 {
			query.WriteString(",")
		}
		query.WriteString(placeholder)
		fsid := file.SId
		if fsid == "" {
			fsid = "*"
		}
		idx := file.Idx
		if discardIdx {
			idx = -1
		}
		args = append(args, hash, idx, file.Name, file.Size, fsid)
		count++
	}

	query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + TableName + ".i = -1 THEN excluded.i ELSE " + TableName + ".i END, s = CASE WHEN " + TableName + ".s = -1 THEN excluded.s ELSE " + TableName + ".s END, sid = CASE WHEN " + TableName + ".sid IN ('', '*') THEN excluded.sid ELSE " + TableName + ".sid END, uat = " + db.CurrentTimestamp)
	_, err := db.Exec(query.String(), args...)
	if err != nil {
		log.Error("failed to track", "error", err)
	}
}

func execBulkTrackFiles(count int, args []any) {
	var query strings.Builder
	query.WriteString("INSERT INTO " + TableName + " (h,i,n,s,sid) VALUES ")

	placeholder := "(?,?,?,?,?)"
	if count > 0 {
		query.WriteString(util.RepeatJoin(placeholder, count, ","))
		query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + TableName + ".i = -1 THEN EXCLUDED.i ELSE " + TableName + ".i END, s = CASE WHEN " + TableName + ".s = -1 THEN EXCLUDED.s ELSE " + TableName + ".s END, sid = CASE WHEN " + TableName + ".sid IN ('', '*') THEN EXCLUDED.sid ELSE " + TableName + ".sid END, uat = " + db.CurrentTimestamp)
		_, err := db.Exec(query.String(), args...)
		if err != nil {
			log.Error("failed to partially bulk track", "error", err)
		}
	}
}

func BulkTrackFiles(filesByHash map[string]Files, discardIdx bool) {
	count := 0
	args := []any{}
	for hash, files := range filesByHash {
		for _, file := range files {
			fsid := file.SId
			if fsid == "" {
				fsid = "*"
			}
			idx := file.Idx
			if discardIdx {
				idx = -1
			}
			args = append(args, hash, idx, file.Name, file.Size, fsid)
			count++
		}
		if count >= 200 {
			execBulkTrackFiles(count, args)
			count = 0
			args = []any{}
		}
	}
	execBulkTrackFiles(count, args)
}

type InsertData struct {
	Hash   string
	Name   string
	Idx    int
	Size   int64
	SId    string
	Source string
}

var record_streams_query_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES ",
	TableName,
	db.JoinColumnNames(
		Column.Hash,
		Column.Name,
		Column.Idx,
		Column.Size,
		Column.SId,
		Column.Source,
	),
)
var record_streams_query_values_placeholder = fmt.Sprintf("(%s)", util.RepeatJoin("?", 6, ","))
var record_streams_query_on_conflict = fmt.Sprintf(
	"ON CONFLICT (%s,%s) DO UPDATE SET %s, %s, %s, %s, uat = ",
	Column.Hash,
	Column.Name,
	fmt.Sprintf("%s = CASE WHEN %s.%s = -1 THEN EXCLUDED.%s ELSE %s.%s END", Column.Idx, TableName, Column.Idx, Column.Idx, TableName, Column.Idx),
	fmt.Sprintf("%s = CASE WHEN %s.%s = -1 THEN EXCLUDED.%s ELSE %s.%s END", Column.Size, TableName, Column.Size, Column.Size, TableName, Column.Size),
	fmt.Sprintf("%s = CASE WHEN %s.%s IN ('', '*') THEN EXCLUDED.%s ELSE %s.%s END", Column.SId, TableName, Column.SId, Column.SId, TableName, Column.SId),
	fmt.Sprintf("%s = EXCLUDED.%s", Column.Source, Column.Source),
)

func Record(items []InsertData) {
	for cItems := range slices.Chunk(items, 200) {
		count := len(cItems)
		args := make([]any, 0, count*6)
		for i := range cItems {
			item := &cItems[i]
			sid := item.SId
			if sid == "" {
				sid = "*"
			}
			args = append(args, item.Hash, item.Name, item.Idx, item.Size, sid, item.Source)
		}
		query := record_streams_query_before_values +
			util.RepeatJoin(record_streams_query_values_placeholder, count, ",") +
			record_streams_query_on_conflict + db.CurrentTimestamp
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed partially to record streams", "error", err)
		}
	}
}

var tag_strem_id_query = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = ? WHERE %s = ? AND %s = ? AND %s IN ('', '*')",
	TableName,
	Column.SId,
	Column.UAt,
	Column.Hash,
	Column.Name,
	Column.SId,
)

func TagStremId(hash string, filename string, sid string) {
	_, err := db.Exec(tag_strem_id_query, sid, db.Timestamp{Time: time.Now()}, hash, filename)
	if err != nil {
		log.Error("failed to tag strem id", "error", err, "hash", hash, "fname", filename, "sid", sid)
	} else {
		log.Debug("tagged strem id", "hash", hash, "fname", filename, "sid", sid)
	}
}
