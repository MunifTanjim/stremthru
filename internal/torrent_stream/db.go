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
	Name   string `json:"n"`
	Idx    int    `json:"i"`
	Size   int64  `json:"s"`
	SId    string `json:"sid,omitempty"`
	Source string `json:"src,omitempty"`
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

func TrackFiles(filesByHash map[string]Files, discardIdx bool) {
	items := []InsertData{}
	for hash, files := range filesByHash {
		for _, file := range files {
			items = append(items, InsertData{Hash: hash, File: file})
		}
	}
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
		Column.Name,
		Column.Idx,
		Column.Size,
		Column.SId,
		Column.Source,
	),
)
var record_streams_query_values_placeholder = fmt.Sprintf("(%s)", util.RepeatJoin("?", 6, ","))
var record_streams_query_on_conflict = fmt.Sprintf(
	" ON CONFLICT (%s,%s) DO UPDATE SET %s, %s, %s, %s, uat = ",
	Column.Hash,
	Column.Name,
	fmt.Sprintf("%s = CASE WHEN ts.%s = -1 THEN EXCLUDED.%s ELSE ts.%s END", Column.Idx, Column.Idx, Column.Idx, Column.Idx),
	fmt.Sprintf("%s = CASE WHEN ts.%s = -1 THEN EXCLUDED.%s ELSE ts.%s END", Column.Size, Column.Size, Column.Size, Column.Size),
	fmt.Sprintf("%s = CASE WHEN ts.%s IN ('', '*') THEN EXCLUDED.%s ELSE ts.%s END", Column.SId, Column.SId, Column.SId, Column.SId),
	fmt.Sprintf("%s = CASE WHEN EXCLUDED.%s != '' THEN EXCLUDED.%s ELSE ts.%s END", Column.Source, Column.Source, Column.Source, Column.Source),
)

func Record(items []InsertData, discardIdx bool) {
	if len(items) == 0 {
		return
	}

	for cItems := range slices.Chunk(items, 200) {
		count := len(cItems)
		args := make([]any, 0, count*6)
		for i := range cItems {
			item := &cItems[i]
			idx := item.Idx
			if discardIdx {
				idx = -1
			}
			sid := item.SId
			if sid == "" {
				sid = "*"
			}
			args = append(args, item.Hash, item.Name, idx, item.Size, sid, item.Source)
		}
		query := record_streams_query_before_values +
			util.RepeatJoin(record_streams_query_values_placeholder, count, ",") +
			record_streams_query_on_conflict + db.CurrentTimestamp
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed partially to record", "error", err)
		} else {
			log.Debug("recorded torrent stream", "count", count)
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

func GetStremIdByHashes(hashes []string) (map[string]string, error) {
	byHash := map[string]string{}
	count := len(hashes)
	if count == 0 {
		return byHash, nil
	}

	query := fmt.Sprintf(
		`SELECT %s, %s FROM %s WHERE %s IN (%s) AND %s like 'tt%%' GROUP BY %s`,
		Column.Hash, Column.SId,
		TableName,
		Column.Hash, util.RepeatJoin("?", count, ","),
		Column.SId,
		Column.Hash,
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
		byHash[hash] = sid
	}

	if err := rows.Err(); err != nil {
		return byHash, err
	}
	return byHash, nil
}
