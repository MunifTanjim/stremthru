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
	TorrentSeasonTypeAbsolute TorrentSeasonType = "abs"
	TorrentSeasonTypeTV       TorrentSeasonType = "tv"
	TorrentSeasonTypeAnime    TorrentSeasonType = "ani"
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

func DeleteTorrentsByHashes(hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}

	for cHashes := range slices.Chunk(hashes, 1000) {
		placeholders := util.RepeatJoin("?", len(cHashes), ",")
		query := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", TorrentTableName, TorrentColumn.Hash, placeholders)
		args := make([]any, len(cHashes))
		for i, h := range cHashes {
			args[i] = h
		}
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed to delete anidb torrent by hashes", "error", err)
			return err
		}
	}

	return nil
}

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

// MappingItem represents a torrent-to-AniDB mapping with enriched data
type MappingItem struct {
	Hash       string            `json:"hash"`
	TTitle     string            `json:"t_title"`
	TId        string            `json:"tid"`
	AniDBTitle string            `json:"anidb_title"`
	SeasonType TorrentSeasonType `json:"s_type"`
	Season     int               `json:"s"`
	EpStart    int               `json:"ep_start"`
	EpEnd      int               `json:"ep_end"`
	UAt        db.Timestamp      `json:"uat"`
}

var query_list_mappings_by_anidb_id_base = fmt.Sprintf(
	`SELECT at.%s, ti.t_title, at.%s, ant.value, at.%s, at.%s, at.%s, at.%s, at.%s
	FROM %s at
	JOIN torrent_info ti ON ti.hash = at.%s
	JOIN %s ant ON ant.%s = at.%s AND ant.%s = 'main'
	WHERE at.%s = ?`,
	TorrentColumn.Hash, TorrentColumn.TId,
	TorrentColumn.SeasonType, TorrentColumn.Season,
	TorrentColumn.EpisodeStart, TorrentColumn.EpisodeEnd, TorrentColumn.UAt,
	TorrentTableName,
	TorrentColumn.Hash,
	TitleTableName, TitleColumn.TId, TorrentColumn.TId, TitleColumn.TType,
	TorrentColumn.TId,
)

var query_list_mappings_by_anidb_id_order_limit = fmt.Sprintf(
	" ORDER BY at.%s DESC, at.%s ASC LIMIT ?",
	TorrentColumn.UAt, TorrentColumn.Hash,
)

var query_list_mappings_by_anidb_id_with_cursor = fmt.Sprintf(
	" AND (at.%s < (SELECT %s FROM %s WHERE %s = ?) OR (at.%s = (SELECT %s FROM %s WHERE %s = ?) AND at.%s > ?))",
	TorrentColumn.UAt, TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash,
	TorrentColumn.UAt, TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash, TorrentColumn.Hash,
)

func ListMappingsByAniDBId(tid string, cursor string, limit int) ([]MappingItem, error) {
	query := query_list_mappings_by_anidb_id_base
	args := []any{tid}

	if cursor != "" {
		query += query_list_mappings_by_anidb_id_with_cursor
		args = append(args, cursor, cursor, cursor)
	}

	query += query_list_mappings_by_anidb_id_order_limit
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Error("failed to list mappings by anidb id", "error", err, "tid", tid)
		return nil, err
	}
	defer rows.Close()

	items := make([]MappingItem, 0, limit)
	for rows.Next() {
		var item MappingItem
		if err := rows.Scan(
			&item.Hash,
			&item.TTitle,
			&item.TId,
			&item.AniDBTitle,
			&item.SeasonType,
			&item.Season,
			&item.EpStart,
			&item.EpEnd,
			&item.UAt,
		); err != nil {
			log.Error("failed to scan mapping item", "error", err)
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// asidToTidExpr returns dialect-specific SQL to extract AniDB ID from torrent_stream.asid
func asidToTidExpr() string {
	switch db.Dialect {
	case db.DBDialectSQLite:
		return "SUBSTR(ts.asid, 1, INSTR(ts.asid || ':', ':') - 1)"
	case db.DBDialectPostgres:
		return "SPLIT_PART(ts.asid, ':', 1)"
	default:
		return "ts.asid"
	}
}

// GetMappingsByHashes returns mapping items for the given hashes
// includeMapped: include items with mappings
// includeUnmapped: include items without mappings
func GetMappingsByHashes(hashes []string, cursor string, limit int, includeMapped, includeUnmapped bool) ([]MappingItem, error) {
	if len(hashes) == 0 {
		return []MappingItem{}, nil
	}

	placeholders := util.RepeatJoin("?", len(hashes), ",")
	asidExpr := asidToTidExpr()

	var query string
	var args []any

	if includeUnmapped && !includeMapped {
		query = fmt.Sprintf(
			`SELECT
				ti.hash,
				ti.t_title,
				'' AS tid,
				'' AS anidb_title,
				'' AS s_type,
				0 AS s,
				0 AS ep_start,
				0 AS ep_end,
				ti.created_at AS uat
			FROM torrent_info ti
			LEFT JOIN %s at ON at.%s = ti.hash
			LEFT JOIN torrent_stream ts ON ts.h = ti.hash AND ts.asid LIKE '%%:%%'
			WHERE ti.hash IN (%s)
				AND (at.%s IS NULL OR at.%s = '')
				AND ts.h IS NULL`,
			TorrentTableName, TorrentColumn.Hash,
			placeholders,
			TorrentColumn.Hash, TorrentColumn.TId,
		)

		args = make([]any, 0, len(hashes)+4)
		for _, h := range hashes {
			args = append(args, h)
		}

		if cursor != "" {
			query += " AND (ti.created_at < (SELECT created_at FROM torrent_info WHERE hash = ?) OR (ti.created_at = (SELECT created_at FROM torrent_info WHERE hash = ?) AND ti.hash > ?))"
			args = append(args, cursor, cursor, cursor)
		}

		query += " ORDER BY ti.created_at DESC, ti.hash ASC LIMIT ?"
	} else if !includeUnmapped && includeMapped {
		query = fmt.Sprintf(
			`SELECT hash, t_title, tid, anidb_title, s_type, s, ep_start, ep_end, uat FROM (
				SELECT at.%s AS hash, ti.t_title, at.%s AS tid, ant.value AS anidb_title, at.%s AS s_type, at.%s AS s, at.%s AS ep_start, at.%s AS ep_end, at.%s AS uat
				FROM %s at
				JOIN torrent_info ti ON ti.hash = at.%s
				JOIN %s ant ON ant.%s = at.%s AND ant.%s = 'main'
				WHERE at.%s IN (%s) AND at.%s != ''

				UNION

				SELECT ts.h AS hash, ti.t_title, ant.%s AS tid, ant.value AS anidb_title, '' AS s_type, 0 AS s, 0 AS ep_start, 0 AS ep_end, ts.uat AS uat
				FROM torrent_stream ts
				JOIN torrent_info ti ON ti.hash = ts.h
				JOIN %s ant ON ant.%s = %s AND ant.%s = 'main'
				WHERE ts.h IN (%s) AND ts.asid LIKE '%%:%%'
			) combined`,
			TorrentColumn.Hash, TorrentColumn.TId,
			TorrentColumn.SeasonType, TorrentColumn.Season,
			TorrentColumn.EpisodeStart, TorrentColumn.EpisodeEnd, TorrentColumn.UAt,
			TorrentTableName,
			TorrentColumn.Hash,
			TitleTableName, TitleColumn.TId, TorrentColumn.TId, TitleColumn.TType,
			TorrentColumn.Hash, placeholders, TorrentColumn.TId,
			TitleColumn.TId,
			TitleTableName, TitleColumn.TId, asidExpr, TitleColumn.TType,
			placeholders,
		)

		args = make([]any, 0, len(hashes)*2+4)
		for _, h := range hashes {
			args = append(args, h)
		}
		for _, h := range hashes {
			args = append(args, h)
		}

		if cursor != "" {
			query += fmt.Sprintf(
				" WHERE (uat < (SELECT %s FROM %s WHERE %s = ?) OR (uat = (SELECT %s FROM %s WHERE %s = ?) AND hash > ?))",
				TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash,
				TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash,
			)
			args = append(args, cursor, cursor, cursor)
		}

		query += " ORDER BY uat DESC, hash ASC LIMIT ?"
	} else {
		query = fmt.Sprintf(
			`SELECT
				ti.hash,
				ti.t_title,
				COALESCE(at.%s, ant_ts.%s, '') AS tid,
				COALESCE(ant_at.value, ant_ts.value, '') AS anidb_title,
				COALESCE(at.%s, '') AS s_type,
				COALESCE(at.%s, 0) AS s,
				COALESCE(at.%s, 0) AS ep_start,
				COALESCE(at.%s, 0) AS ep_end,
				COALESCE(at.%s, ts.uat, ti.created_at) AS uat
			FROM torrent_info ti
			LEFT JOIN %s at ON at.%s = ti.hash
			LEFT JOIN %s ant_at ON ant_at.%s = at.%s AND ant_at.%s = 'main'
			LEFT JOIN torrent_stream ts ON ts.h = ti.hash AND ts.asid LIKE '%%:%%'
			LEFT JOIN %s ant_ts ON ant_ts.%s = %s AND ant_ts.%s = 'main'
			WHERE ti.hash IN (%s)`,
			TorrentColumn.TId, TitleColumn.TId,
			TorrentColumn.SeasonType,
			TorrentColumn.Season,
			TorrentColumn.EpisodeStart,
			TorrentColumn.EpisodeEnd,
			TorrentColumn.UAt,
			TorrentTableName, TorrentColumn.Hash,
			TitleTableName, TitleColumn.TId, TorrentColumn.TId, TitleColumn.TType,
			TitleTableName, TitleColumn.TId, asidExpr, TitleColumn.TType,
			placeholders,
		)

		args = make([]any, 0, len(hashes)+4)
		for _, h := range hashes {
			args = append(args, h)
		}

		if cursor != "" {
			query += fmt.Sprintf(
				" AND (COALESCE(at.%s, ts.uat, ti.created_at) < (SELECT COALESCE(at2.%s, ti2.created_at) FROM torrent_info ti2 LEFT JOIN %s at2 ON at2.%s = ti2.hash WHERE ti2.hash = ?) OR (COALESCE(at.%s, ts.uat, ti.created_at) = (SELECT COALESCE(at2.%s, ti2.created_at) FROM torrent_info ti2 LEFT JOIN %s at2 ON at2.%s = ti2.hash WHERE ti2.hash = ?) AND ti.hash > ?))",
				TorrentColumn.UAt, TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash,
				TorrentColumn.UAt, TorrentColumn.UAt, TorrentTableName, TorrentColumn.Hash,
			)
			args = append(args, cursor, cursor, cursor)
		}

		query += fmt.Sprintf(" ORDER BY COALESCE(at.%s, ts.uat, ti.created_at) DESC, ti.hash ASC LIMIT ?", TorrentColumn.UAt)

	}

	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Error("failed to get mappings by hashes", "error", err)
		return nil, err
	}
	defer rows.Close()

	items := make([]MappingItem, 0, limit)
	for rows.Next() {
		var item MappingItem
		if err := rows.Scan(
			&item.Hash,
			&item.TTitle,
			&item.TId,
			&item.AniDBTitle,
			&item.SeasonType,
			&item.Season,
			&item.EpStart,
			&item.EpEnd,
			&item.UAt,
		); err != nil {
			log.Error("failed to scan mapping item", "error", err)
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
