package anidb

import (
	"fmt"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TorrentTableName = "anidb_torrent"

type AniDBTorrent struct {
	TId  string       `json:"tid"`
	Hash string       `json:"hash"`
	UAt  db.Timestamp `json:"uat"`
}

var TorrentColumn = struct {
	TId  string
	Hash string
	UAt  string
}{
	TId:  "tid",
	Hash: "hash",
	UAt:  "uat",
}

var query_insert_torrents_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s, %s) VALUES ",
	TorrentTableName,
	TorrentColumn.TId,
	TorrentColumn.Hash,
)
var query_insert_torrents_values_placeholder = "(?,?)"
var query_insert_torrents_after_values = fmt.Sprintf(
	" ON CONFLICT (%s, %s) DO UPDATE SET %s = %s",
	TorrentColumn.TId,
	TorrentColumn.Hash,
	TorrentColumn.UAt,
	db.CurrentTimestamp,
)

func InsertTorrents(items []AniDBTorrent) error {
	if len(items) == 0 {
		return nil
	}

	for cItems := range slices.Chunk(items, 1000) {
		count := len(cItems)
		args := make([]any, count*2)
		for i, item := range cItems {
			args[i*2+0] = item.TId
			args[i*2+1] = item.Hash
		}

		query := query_insert_torrents_before_values + util.RepeatJoin(query_insert_torrents_values_placeholder, count, ",") + query_insert_torrents_after_values
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed to insert anidb torrent", "error", err)
			return err
		} else {
			log.Debug("inserted anidb torrent", "count", count)
		}
	}

	return nil
}
