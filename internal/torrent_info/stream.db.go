package torrent_info

import (
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type TorrentStream struct {
	Hash     string            `json:"hash"`
	SId      string            `json:"sid"`
	Source   TorrentInfoSource `json:"src"`
	FileIdx  int               `json:"f_idx"`
	FileName string            `json:"f_name"`
	FileSize int64             `json:"f_size"`
	CAt      db.Timestamp      `json:"cat"`
	UAt      db.Timestamp      `json:"uat"`
}

const StreamTableName = "torrent_stream"

type StreamColumnStruct struct {
	Hash     string
	SId      string
	Source   string
	FileIdx  string
	FileName string
	FileSize string
	CAt      string
	UAt      string
}

var StreamColumn = StreamColumnStruct{
	Hash:     "hash",
	SId:      "sid",
	Source:   "src",
	FileIdx:  "f_idx",
	FileName: "f_name",
	FileSize: "f_size",
	CAt:      "cat",
	UAt:      "uat",
}

var StreamColumns = []string{
	StreamColumn.Hash,
	StreamColumn.SId,
	StreamColumn.Source,
	StreamColumn.FileIdx,
	StreamColumn.FileName,
	StreamColumn.FileSize,
	StreamColumn.CAt,
	StreamColumn.UAt,
}

var insert_stream_query_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s,%s,%s,%s,%s,%s) VALUES ",
	StreamTableName,
	StreamColumn.Hash,
	StreamColumn.SId,
	StreamColumn.Source,
	StreamColumn.FileIdx,
	StreamColumn.FileName,
	StreamColumn.FileSize,
)
var insert_stream_query_values_placeholder = "(?,?,?,?,?,?)"
var insert_stream_query_after_values = fmt.Sprintf(
	" ON CONFLICT (%s,%s) DO UPDATE SET %s = EXCLUDED.%s, %s = ",
	StreamColumn.Hash,
	StreamColumn.SId,
	StreamColumn.Source,
	StreamColumn.Source,
	StreamColumn.UAt,
)

func get_insert_stream_query(count int) string {
	return insert_stream_query_before_values + util.RepeatJoin(insert_stream_query_values_placeholder, count, ",") + insert_stream_query_after_values + db.CurrentTimestamp
}

type TorrentStreamInsertData struct {
	Hash string
	File TorrentInfoInsertDataFile
}

func InsertStreams(hashesBySource map[TorrentInfoSource][]TorrentStreamInsertData, sid string) {
	args := []any{}
	for src, items := range hashesBySource {
		for _, item := range items {
			args = append(args, item.Hash, sid, src, item.File.Idx, item.File.Name, item.File.Size)
		}
	}
	query := get_insert_stream_query(len(args) / 6)
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Error("failed to insert torrent stream", "error", err)
	}
}
