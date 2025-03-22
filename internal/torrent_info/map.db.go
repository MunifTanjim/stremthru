package torrent_info

import (
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type TorrentInfoMap struct {
	Hash   string            `json:"hash"`
	SId    string            `json:"sid"`
	Source TorrentInfoSource `json:"src"`
	CAt    db.Timestamp      `json:"cat"`
	UAt    db.Timestamp      `json:"uat"`
}

const MapTableName = "torrent_info_map"

type MapColumnStruct struct {
	Hash   string
	SId    string
	Source string
	CAt    string
	UAt    string
}

var MapColumn = MapColumnStruct{
	Hash:   "hash",
	SId:    "sid",
	Source: "src",
	CAt:    "cat",
	UAt:    "uat",
}

var MapColumns = []string{
	MapColumn.Hash,
	MapColumn.SId,
	MapColumn.Source,
	MapColumn.CAt,
	MapColumn.UAt,
}

var record_map_query_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s,%s,%s) VALUES ",
	MapTableName,
	MapColumn.Hash,
	MapColumn.SId,
	MapColumn.Source,
)
var record_map_query_values_placeholder = "(?,?,?)"
var record_map_query_after_values = fmt.Sprintf(
	" ON CONFLICT (%s,%s) DO UPDATE SET %s = EXCLUDED.%s, %s = ",
	MapColumn.Hash,
	MapColumn.SId,
	MapColumn.Source,
	MapColumn.Source,
	MapColumn.UAt,
)

func get_record_map_query(count int) string {
	return record_map_query_before_values + util.RepeatJoin(record_map_query_values_placeholder, count, ",") + record_map_query_after_values + db.CurrentTimestamp
}

func RecordMap(hashesBySource map[TorrentInfoSource][]string, sid string) {
	args := []any{}
	for src, hashes := range hashesBySource {
		for _, hash := range hashes {
			args = append(args, hash, sid, src)
		}
	}
	query := get_record_map_query(len(args) / 3)
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Error("failed to record torrent info map", "error", err)
	}
}
