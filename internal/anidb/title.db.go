package anidb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TitleTableName = "anidb_title"

type AniDBTitle struct {
	Id     int
	TId    string
	TType  string
	TLang  string
	Value  string
	Season string
	Year   string
}

var TitleColumn = struct {
	Id     string
	TId    string
	TType  string
	TLang  string
	Value  string
	Season string
	Year   string
}{
	Id:     "id",
	TId:    "tid",
	TType:  "ttype",
	TLang:  "tlang",
	Value:  "value",
	Season: "season",
	Year:   "year",
}

var sl_query_rebuild_title_fts = fmt.Sprintf(
	`INSERT INTO %s_fts(%s_fts) VALUES('rebuild')`,
	TitleTableName,
	TitleTableName,
)

func sqliteRebuildTitleFTS() error {
	_, err := db.Exec(sl_query_rebuild_title_fts)
	return err
}

func postgresRebuildTitleFTS() error {
	return nil
}

var RebuildTitleFTS = func() func() error {
	if db.Dialect == db.DBDialectSQLite {
		return sqliteRebuildTitleFTS
	}
	return postgresRebuildTitleFTS
}()

var query_upsert_titles_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TitleTableName,
	strings.Join([]string{
		TitleColumn.TId,
		TitleColumn.TType,
		TitleColumn.TLang,
		TitleColumn.Value,
		TitleColumn.Season,
		TitleColumn.Year,
	}, ","),
)
var query_upsert_titles_values_placeholder = "(" + util.RepeatJoin("?", 6, ",") + ")"
var query_upsert_titles_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s,%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s`,
	TitleColumn.TId,
	TitleColumn.TType,
	TitleColumn.TLang,
	TitleColumn.Value,
	TitleColumn.Value,
	TitleColumn.Season,
	TitleColumn.Season,
	TitleColumn.Year,
	TitleColumn.Year,
)

func UpsertTitles(titles []AniDBTitle) error {
	count := len(titles)
	if count == 0 {
		return nil
	}
	query := query_upsert_titles_before_values +
		util.RepeatJoin(query_upsert_titles_values_placeholder, count, ",") +
		query_upsert_titles_after_values
	var args []any
	for _, t := range titles {
		args = append(args, t.TId, t.TType, t.TLang, t.Value, t.Season, t.Year)
	}

	_, err := db.Exec(query, args...)
	return err
}

var sl_query_search_ids_by_title_select = fmt.Sprintf(
	"SELECT DISTINCT at.%s FROM %s_fts(?) atf JOIN %s at ON at.rowid = atf.rowid WHERE rank = 'bm25(10,10)' ORDER BY ",
	TitleColumn.TId,
	TitleTableName,
	TitleTableName,
)
var sl_query_search_ids_by_title_order_by_cond_year_start = fmt.Sprintf(
	` CASE WHEN atf.%s = ?`,
	TitleColumn.Year,
)
var sl_query_search_ids_by_title_order_by_cond_season_start = fmt.Sprintf(
	` CASE WHEN atf.%s IN `,
	TitleColumn.Season,
)
var query_search_ids_by_title_order_by_cond_value_end = " THEN 0 ELSE 1 END, "
var sl_query_search_ids_by_title_order_by_common = fmt.Sprintf(
	" CASE WHEN lower(atf.%s) = ? THEN 0 ELSE 1 END, rank LIMIT ?",
	TitleColumn.Value,
)

func sqliteSearchIdsByTitle(title string, seasons []int, year int, limit int) ([]string, error) {
	title = strings.ToLower(title)

	fts_query := title
	fts_query = db.PrepareFTS5Query(fts_query)
	if fts_query == "" {
		return []string{}, nil
	}

	var query strings.Builder
	var args []any

	query.WriteString(sl_query_search_ids_by_title_select)
	args = append(args, fts_query)

	if year != 0 {
		query.WriteString(sl_query_search_ids_by_title_order_by_cond_year_start)
		args = append(args, strconv.Itoa(year))
		query.WriteString(query_search_ids_by_title_order_by_cond_value_end)
	}

	seasonCount := len(seasons)
	if seasonCount > 0 {
		query.WriteString(sl_query_search_ids_by_title_order_by_cond_season_start)
		query.WriteString("(" + util.RepeatJoin("?", seasonCount, ",") + ")")
		for _, s := range seasons {
			args = append(args, strconv.Itoa(s))
		}
		query.WriteString(query_search_ids_by_title_order_by_cond_value_end)
	}

	query.WriteString(sl_query_search_ids_by_title_order_by_common)
	args = append(args, title)

	if limit == 0 {
		if seasonCount > 0 {
			limit = seasonCount
		} else {
			limit = 1
		}
	}
	args = append(args, limit)

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

var pg_query_search_ids_by_title_select = fmt.Sprintf(
	"SELECT DISTINCT %s FROM %s WHERE search_vector @@ plainto_tsquery('simple', ?) ORDER BY ",
	TitleColumn.TId,
	TitleTableName,
)
var pg_query_search_ids_by_title_order_by_cond_year_start = fmt.Sprintf(
	` CASE WHEN %s = ?`,
	TitleColumn.Year,
)
var pg_query_search_ids_by_title_order_by_cond_season_start = fmt.Sprintf(
	` CASE WHEN %s IN `,
	TitleColumn.Season,
)
var pg_query_search_ids_by_title_order_by_common = fmt.Sprintf(
	" CASE WHEN lower(%s) = ? THEN 0 ELSE 1 END, -ts_rank(search_vector, plainto_tsquery('simple', ?)) LIMIT ?",
	TitleColumn.Value,
)

func postgresSearchIdsByTitle(title string, seasons []int, year int, limit int) ([]string, error) {
	title = strings.ToLower(title)

	fts_query := title
	if fts_query == "" {
		return []string{}, nil
	}

	var query strings.Builder
	var args []any

	query.WriteString(pg_query_search_ids_by_title_select)
	args = append(args, fts_query)

	if year != 0 {
		query.WriteString(pg_query_search_ids_by_title_order_by_cond_year_start)
		args = append(args, strconv.Itoa(year))
		query.WriteString(query_search_ids_by_title_order_by_cond_value_end)
	}

	seasonCount := len(seasons)
	if seasonCount != 0 {
		query.WriteString(pg_query_search_ids_by_title_order_by_cond_season_start)
		query.WriteString("(" + util.RepeatJoin("?", seasonCount, ",") + ")")
		for _, s := range seasons {
			args = append(args, strconv.Itoa(s))
		}
		query.WriteString(query_search_ids_by_title_order_by_cond_value_end)
	}

	query.WriteString(pg_query_search_ids_by_title_order_by_common)
	args = append(args, title, fts_query)

	if limit == 0 {
		if seasonCount > 0 {
			limit = seasonCount
		} else {
			limit = 1
		}
	}
	args = append(args, limit)

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		ids = append(ids, tid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

var SearchIdsByTitle = func() func(title string, seasons []int, year int, limit int) ([]string, error) {
	if db.Dialect == db.DBDialectSQLite {
		return sqliteSearchIdsByTitle
	}
	return postgresSearchIdsByTitle
}()

var query_get_titles_by_ids = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s IN `,
	strings.Join([]string{
		TitleColumn.TId,
		TitleColumn.TType,
		TitleColumn.TLang,
		TitleColumn.Value,
		TitleColumn.Season,
		TitleColumn.Year,
	}, ","),
	TitleTableName,
	TitleColumn.TId,
)

func GetTitlesByIds(anidbIds []string) ([]AniDBTitle, error) {
	if len(anidbIds) == 0 {
		return []AniDBTitle{}, nil
	}

	query := query_get_titles_by_ids + "(" + util.RepeatJoin("?", len(anidbIds), ",") + ")"
	args := make([]any, len(anidbIds))
	for i, id := range anidbIds {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := []AniDBTitle{}
	for rows.Next() {
		var t AniDBTitle
		if err := rows.Scan(
			&t.TId,
			&t.TType,
			&t.TLang,
			&t.Value,
			&t.Season,
			&t.Year,
		); err != nil {
			return nil, err
		}
		titles = append(titles, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titles, nil
}
