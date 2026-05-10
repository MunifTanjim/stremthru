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

var query_get_last_mapped_imdb_id = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s > '' ORDER BY %s DESC LIMIT 1;",
	Column.TId,
	TableName,
	Column.TId,
	Column.UAt,
)

func GetLastMappedIMDBId() (string, error) {
	var lastIMDBId string
	row := db.QueryRow(query_get_last_mapped_imdb_id)
	err := row.Scan(&lastIMDBId)
	return lastIMDBId, err
}

func DeleteByHashes(hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}

	for cHashes := range slices.Chunk(hashes, 1000) {
		placeholders := util.RepeatJoin("?", len(cHashes), ",")
		query := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", TableName, Column.Hash, placeholders)
		args := make([]any, len(cHashes))
		for i, h := range cHashes {
			args[i] = h
		}
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed to delete imdb torrent by hashes", "error", err)
			return err
		}
	}

	return nil
}

// MappingItem represents a torrent-to-IMDB mapping with enriched data
type MappingItem struct {
	Hash      string       `json:"hash"`
	TTitle    string       `json:"t_title"`
	TId       string       `json:"tid"`
	IMDBTitle string       `json:"imdb_title"`
	IMDBYear  int          `json:"imdb_year"`
	IMDBType  string       `json:"imdb_type"`
	UAt       db.Timestamp `json:"uat"`
}

// sidToTidExpr returns dialect-specific SQL to extract IMDB ID from torrent_stream.sid
func sidToTidExpr() string {
	switch db.Dialect {
	case db.DBDialectSQLite:
		return "SUBSTR(ts.sid, 1, INSTR(ts.sid || ':', ':') - 1)"
	case db.DBDialectPostgres:
		return "SPLIT_PART(ts.sid, ':', 1)"
	default:
		return "ts.sid"
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
	sidExpr := sidToTidExpr()

	var query string
	var args []any

	if includeUnmapped && !includeMapped {
		query = fmt.Sprintf(
			`SELECT
				ti.hash,
				ti.t_title,
				'' AS tid,
				'' AS imdb_title,
				0 AS imdb_year,
				'' AS imdb_type,
				ti.created_at AS uat
			FROM torrent_info ti
			LEFT JOIN %s it ON it.%s = ti.hash
			LEFT JOIN torrent_stream ts ON ts.h = ti.hash AND ts.sid LIKE 'tt%%'
			WHERE ti.hash IN (%s)
				AND (it.%s IS NULL OR it.%s = '')
				AND ts.h IS NULL`,
			TableName, Column.Hash,
			placeholders,
			Column.Hash, Column.TId,
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
			`SELECT hash, t_title, tid, imdb_title, imdb_year, imdb_type, uat FROM (
				SELECT ti.hash, ti.t_title, it.%s AS tid, imt.title AS imdb_title, imt.year AS imdb_year, imt.type AS imdb_type, it.%s AS uat
				FROM torrent_info ti
				JOIN %s it ON it.%s = ti.hash AND it.%s != ''
				JOIN imdb_title imt ON imt.tid = it.%s
				WHERE ti.hash IN (%s)

				UNION

				SELECT ti.hash, ti.t_title, imt.tid AS tid, imt.title AS imdb_title, imt.year AS imdb_year, imt.type AS imdb_type, ts.uat AS uat
				FROM torrent_info ti
				JOIN torrent_stream ts ON ts.h = ti.hash AND ts.sid LIKE 'tt%%'
				JOIN imdb_title imt ON imt.tid = %s
				WHERE ti.hash IN (%s)
			) combined`,
			Column.TId, Column.UAt,
			TableName, Column.Hash, Column.TId,
			Column.TId,
			placeholders,
			sidExpr,
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
				Column.UAt, TableName, Column.Hash,
				Column.UAt, TableName, Column.Hash,
			)
			args = append(args, cursor, cursor, cursor)
		}

		query += " ORDER BY uat DESC, hash ASC LIMIT ?"
	} else {
		query = fmt.Sprintf(
			`SELECT
				ti.hash,
				ti.t_title,
				COALESCE(it.%s, imt_ts.tid, '') AS tid,
				COALESCE(imt_it.title, imt_ts.title, '') AS imdb_title,
				COALESCE(imt_it.year, imt_ts.year, 0) AS imdb_year,
				COALESCE(imt_it.type, imt_ts.type, '') AS imdb_type,
				COALESCE(it.%s, ts.uat, ti.created_at) AS uat
			FROM torrent_info ti
			LEFT JOIN %s it ON it.%s = ti.hash
			LEFT JOIN imdb_title imt_it ON imt_it.tid = it.%s
			LEFT JOIN torrent_stream ts ON ts.h = ti.hash AND ts.sid LIKE 'tt%%'
			LEFT JOIN imdb_title imt_ts ON imt_ts.tid = %s
			WHERE ti.hash IN (%s)`,
			Column.TId,
			Column.UAt,
			TableName, Column.Hash,
			Column.TId,
			sidExpr,
			placeholders,
		)

		args = make([]any, 0, len(hashes)+4)
		for _, h := range hashes {
			args = append(args, h)
		}

		if cursor != "" {
			query += fmt.Sprintf(
				" AND (COALESCE(it.%s, ts.uat, ti.created_at) < (SELECT COALESCE(it2.%s, ti2.created_at) FROM torrent_info ti2 LEFT JOIN %s it2 ON it2.%s = ti2.hash WHERE ti2.hash = ?) OR (COALESCE(it.%s, ts.uat, ti.created_at) = (SELECT COALESCE(it2.%s, ti2.created_at) FROM torrent_info ti2 LEFT JOIN %s it2 ON it2.%s = ti2.hash WHERE ti2.hash = ?) AND ti.hash > ?))",
				Column.UAt, Column.UAt, TableName, Column.Hash,
				Column.UAt, Column.UAt, TableName, Column.Hash,
			)
			args = append(args, cursor, cursor, cursor)
		}

		query += fmt.Sprintf(" ORDER BY COALESCE(it.%s, ts.uat, ti.created_at) DESC, ti.hash ASC LIMIT ?", Column.UAt)

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
			&item.IMDBTitle,
			&item.IMDBYear,
			&item.IMDBType,
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
