package imdb_torrent

import (
	"fmt"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TableName = "imdb_torrent"

type IMDBTorrent struct {
	TId  string       `json:"tid"`
	Hash string       `json:"hash"`
	UAt  db.Timestamp `json:"uat"`
}

type ColumnStruct struct {
	TId  string
	Hash string
	UAt  string
}

var Column = ColumnStruct{
	TId:  "tid",
	Hash: "hash",
	UAt:  "uat",
}

var query_insert_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s, %s) VALUES ",
	TableName,
	Column.Hash,
	Column.TId,
)
var query_insert_values_placeholder = "(?,?)"
var query_insert_after_values = fmt.Sprintf(
	" ON CONFLICT (%s, %s) DO UPDATE SET %s = %s",
	Column.Hash,
	Column.TId,
	Column.UAt,
	db.CurrentTimestamp,
)

func Insert(items []IMDBTorrent) error {
	if len(items) == 0 {
		return nil
	}

	for cItems := range slices.Chunk(items, 1000) {
		count := len(cItems)
		args := make([]any, count*2)
		for i, item := range cItems {
			args[i*2] = item.Hash
			args[i*2+1] = item.TId
		}

		query := query_insert_before_values + util.RepeatJoin(query_insert_values_placeholder, count, ",") + query_insert_after_values
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed to insert imdb torrent", "error", err)
			return err
		} else {
			log.Debug("inserted imdb torrent", "count", count)
		}
	}

	return nil
}
