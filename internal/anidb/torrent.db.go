package anidb

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TorrentTableName = "anidb_torrent"

type TorrentSeasonType string

const (
	TorrentSeasonTypeAbsolute  TorrentSeasonType = "abs"
	TorrentSeasonTypeTV        TorrentSeasonType = "tv"
	TorrentSeasonTypeTypeAnime TorrentSeasonType = "ani"
)

type AniDBTorrent struct {
	TId          string               `json:"tid"`
	Hash         string               `json:"hash"`
	SeasonType   TorrentSeasonType    `json:"s_type"`
	Season       int                  `json:"s"`
	EpisodeStart int                  `json:"ep_start"`
	EpisodeEnd   int                  `json:"ep_end"`
	Episodes     db.CommaSeperatedInt `json:"eps"`
	UAt          db.Timestamp         `json:"uat"`
}

var TorrentColumn = struct {
	TId          string
	Hash         string
	SeasonType   string
	Season       string
	EpisodeStart string
	EpisodeEnd   string
	Episodes     string
	UAt          string
}{
	TId:          "tid",
	Hash:         "hash",
	SeasonType:   "s_type",
	Season:       "s",
	EpisodeStart: "ep_start",
	EpisodeEnd:   "ep_end",
	Episodes:     "eps",
	UAt:          "uat",
}

var TorrentColumns = []string{
	TorrentColumn.TId,
	TorrentColumn.Hash,
	TorrentColumn.SeasonType,
	TorrentColumn.Season,
	TorrentColumn.EpisodeStart,
	TorrentColumn.EpisodeEnd,
	TorrentColumn.Episodes,
	TorrentColumn.UAt,
}

var query_upsert_torrents_before_values = fmt.Sprintf(
	"INSERT INTO %s (%s) VALUES ",
	TorrentTableName,
	strings.Join([]string{
		TorrentColumn.TId,
		TorrentColumn.Hash,
		TorrentColumn.SeasonType,
		TorrentColumn.Season,
		TorrentColumn.EpisodeStart,
		TorrentColumn.EpisodeEnd,
		TorrentColumn.Episodes,
	}, ","),
)
var query_upsert_torrents_values_placeholder = "(" + util.RepeatJoin("?", len(TorrentColumns)-1, ",") + ")"
var query_upsert_torrents_after_values = fmt.Sprintf(
	" ON CONFLICT (%s) DO UPDATE SET %s",
	strings.Join([]string{
		TorrentColumn.TId,
		TorrentColumn.Hash,
		TorrentColumn.SeasonType,
		TorrentColumn.Season,
	}, ","),
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, TorrentColumn.EpisodeStart, TorrentColumn.EpisodeStart),
		fmt.Sprintf(`%s = EXCLUDED.%s`, TorrentColumn.EpisodeEnd, TorrentColumn.EpisodeEnd),
		fmt.Sprintf(`%s = EXCLUDED.%s`, TorrentColumn.Episodes, TorrentColumn.Episodes),
		fmt.Sprintf(`%s = %s`, TorrentColumn.UAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertTorrents(items []AniDBTorrent) error {
	if len(items) == 0 {
		return nil
	}

	columnCount := len(TorrentColumns) - 1
	for cItems := range slices.Chunk(items, 500) {
		count := len(cItems)
		args := make([]any, count*columnCount)
		for i, item := range cItems {
			idx := i * columnCount
			args[idx+0] = item.TId
			args[idx+1] = item.Hash
			args[idx+2] = item.SeasonType
			args[idx+3] = item.Season
			args[idx+4] = item.EpisodeStart
			args[idx+5] = item.EpisodeEnd
			args[idx+6] = item.Episodes
		}

		query := query_upsert_torrents_before_values + util.RepeatJoin(query_upsert_torrents_values_placeholder, count, ",") + query_upsert_torrents_after_values
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
