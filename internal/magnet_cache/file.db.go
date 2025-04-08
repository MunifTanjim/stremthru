package magnet_cache

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const FileTableName = "magnet_cache_file"

var mcfLog = logger.Scoped(FileTableName)

type MagnetCacheFile struct {
	Hash string `json:"-"`
	Name string `json:"n"`
	Idx  int    `json:"i"`
	Size int64  `json:"s"`
	SId  string `json:"-"`
}

type MCFile struct {
	Hash   string       `json:"h"`
	Name   string       `json:"n"`
	Idx    int          `json:"i"`
	Size   int64        `json:"s"`
	SId    string       `json:"sid"`
	Source string       `json:"src"`
	CAt    db.Timestamp `json:"cat"`
	UAt    db.Timestamp `json:"uat"`
	SAt    db.Timestamp `json:"sat,omitzero"`
}

type MCFileColumnStruct struct {
	Hash   string
	Name   string
	Idx    string
	Size   string
	SId    string
	Source string
	CAt    string
	UAt    string
	SAt    string
}

var MCFileColumn = MCFileColumnStruct{
	Hash:   "h",
	Name:   "n",
	Idx:    "i",
	Size:   "s",
	SId:    "sid",
	Source: "src",
	CAt:    "cat",
	UAt:    "uat",
	SAt:    "sat",
}

var MCFileColumns = []string{
	MCFileColumn.Hash,
	MCFileColumn.SId,
	MCFileColumn.Name,
	MCFileColumn.Idx,
	MCFileColumn.Size,
	MCFileColumn.Source,
	MCFileColumn.CAt,
	MCFileColumn.UAt,
	MCFileColumn.SAt,
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

	rows, err := db.Query("SELECT h, "+db.FnJSONGroupArray+"("+db.FnJSONObject+"('i', i, 'n', n, 's', s)) AS files FROM "+FileTableName+" WHERE h IN ("+strings.Join(hashPlaceholders, ",")+") GROUP BY h", args...)
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
	query.WriteString("INSERT INTO " + FileTableName + " (h,i,n,s,sid) VALUES ")
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

	query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + FileTableName + ".i = -1 THEN excluded.i ELSE " + FileTableName + ".i END, s = CASE WHEN " + FileTableName + ".s = -1 THEN excluded.s ELSE " + FileTableName + ".s END, sid = CASE WHEN " + FileTableName + ".sid IN ('', '*') THEN excluded.sid ELSE " + FileTableName + ".sid END, uat = " + db.CurrentTimestamp)
	_, err := db.Exec(query.String(), args...)
	if err != nil {
		mcfLog.Error("failed to track", "error", err)
	}
}

func execBulkTrackFiles(count int, args []any) {
	var query strings.Builder
	query.WriteString("INSERT INTO " + FileTableName + " (h,i,n,s,sid) VALUES ")

	placeholder := "(?,?,?,?,?)"
	if count > 0 {
		query.WriteString(util.RepeatJoin(placeholder, count, ","))
		query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + FileTableName + ".i = -1 THEN EXCLUDED.i ELSE " + FileTableName + ".i END, s = CASE WHEN " + FileTableName + ".s = -1 THEN EXCLUDED.s ELSE " + FileTableName + ".s END, sid = CASE WHEN " + FileTableName + ".sid IN ('', '*') THEN EXCLUDED.sid ELSE " + FileTableName + ".sid END, uat = " + db.CurrentTimestamp)
		_, err := db.Exec(query.String(), args...)
		if err != nil {
			mcfLog.Error("failed to partially bulk track", "error", err)
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

type MCFileInsertData struct {
	Hash   string
	Name   string
	Idx    int
	Size   int64
	SId    string
	Source string
}

var record_streams_query_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES ",
	FileTableName,
	db.JoinColumnNames(
		MCFileColumn.Hash,
		MCFileColumn.Name,
		MCFileColumn.Idx,
		MCFileColumn.Size,
		MCFileColumn.SId,
		MCFileColumn.Source,
	),
)
var record_streams_query_values_placeholder = fmt.Sprintf("(%s)", util.RepeatJoin("?", 6, ","))
var record_streams_query_on_conflict = fmt.Sprintf(
	"ON CONFLICT (%s,%s) DO UPDATE SET %s, %s, %s, %s, uat = ",
	MCFileColumn.Hash,
	MCFileColumn.Name,
	fmt.Sprintf("%s = CASE WHEN %s.%s = -1 THEN EXCLUDED.%s ELSE %s.%s END", MCFileColumn.Idx, FileTableName, MCFileColumn.Idx, MCFileColumn.Idx, FileTableName, MCFileColumn.Idx),
	fmt.Sprintf("%s = CASE WHEN %s.%s = -1 THEN EXCLUDED.%s ELSE %s.%s END", MCFileColumn.Size, FileTableName, MCFileColumn.Size, MCFileColumn.Size, FileTableName, MCFileColumn.Size),
	fmt.Sprintf("%s = CASE WHEN %s.%s IN ('', '*') THEN EXCLUDED.%s ELSE %s.%s END", MCFileColumn.SId, FileTableName, MCFileColumn.SId, MCFileColumn.SId, FileTableName, MCFileColumn.SId),
	fmt.Sprintf("%s = EXCLUDED.%s", MCFileColumn.Source, MCFileColumn.Source),
)

func RecordStreams(items []MCFileInsertData) {
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
			mcfLog.Error("failed partially to record streams", "error", err)
		}
	}
}
