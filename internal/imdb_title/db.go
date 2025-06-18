package imdb_title

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type IMDBTitleType string

const (
	IMDBTitleTypeShort        IMDBTitleType = "short"
	IMDBTitleTypeMovie        IMDBTitleType = "movie"
	IMDBTitleTypeTvShort      IMDBTitleType = "tvShort"
	IMDBTitleTypeTvMovie      IMDBTitleType = "tvMovie"
	IMDBTitleTypeTvEpisode    IMDBTitleType = "tvEpisode"
	IMDBTitleTypeTvSeries     IMDBTitleType = "tvSeries"
	IMDBTitleTypeTvMiniSeries IMDBTitleType = "tvMiniSeries"
	IMDBTitleTypeTvSpecial    IMDBTitleType = "tvSpecial"
	IMDBTitleTypeVideo        IMDBTitleType = "video"
	IMDBTitleTypeVideoGame    IMDBTitleType = "videoGame"
)

type IMDBTitle struct {
	Id        int    `json:"-"`
	TId       string `json:"tid"`
	Title     string `json:"title"`
	OrigTitle string `json:"orig_title"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	IsAdult   bool   `json:"is_adult"`
}

const TableName = "imdb_title"

type ColumnStruct struct {
	Id        string
	TId       string
	Title     string
	OrigTitle string
	Year      string
	Type      string
	IsAdult   string
}

var Column = ColumnStruct{
	Id:        "id",
	TId:       "tid",
	Title:     "title",
	OrigTitle: "orig_title",
	Year:      "year",
	Type:      "type",
	IsAdult:   "is_adult",
}

var query_get = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join([]string{
		Column.Id,
		Column.TId,
		Column.Title,
		Column.OrigTitle,
		Column.Year,
		Column.Type,
		Column.IsAdult,
	}, ","),
	TableName,
	Column.TId,
)

func Get(tid string) (*IMDBTitle, error) {
	row := db.QueryRow(query_get, tid)
	var title IMDBTitle
	if err := row.Scan(
		&title.Id,
		&title.TId,
		&title.Title,
		&title.OrigTitle,
		&title.Year,
		&title.Type,
		&title.IsAdult,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if title.OrigTitle == "" {
		title.OrigTitle = title.Title
	}

	return &title, nil
}

var query_by_ids_prefix = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s IN `,
	strings.Join([]string{
		Column.Id,
		Column.TId,
		Column.Title,
		Column.OrigTitle,
		Column.Year,
		Column.Type,
		Column.IsAdult,
	}, ","),
	TableName,
	Column.TId,
)

func ListByIds(tids []string) ([]IMDBTitle, error) {
	query := fmt.Sprintf("%s (%s)", query_by_ids_prefix, strings.Repeat("?,", len(tids)-1)+"?")
	args := make([]any, len(tids))
	for i, id := range tids {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []IMDBTitle
	for rows.Next() {
		var title IMDBTitle
		if err := rows.Scan(
			&title.Id,
			&title.TId,
			&title.Title,
			&title.OrigTitle,
			&title.Year,
			&title.Type,
			&title.IsAdult,
		); err != nil {
			return nil, err
		}

		if title.OrigTitle == "" {
			title.OrigTitle = title.Title
		}

		titles = append(titles, title)
	}

	return titles, nil
}

var query_get_type_by_ids = fmt.Sprintf(
	`SELECT %s, %s FROM %s WHERE %s IN `,
	Column.TId,
	Column.Type,
	TableName,
	Column.TId,
)

func GetTypeByIds(tids []string) (map[string]IMDBTitleType, error) {
	count := len(tids)
	typeMap := make(map[string]IMDBTitleType, count)

	if count == 0 {
		return typeMap, nil
	}

	query := query_get_type_by_ids + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i, id := range tids {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tid string
		var titleType string
		if err := rows.Scan(&tid, &titleType); err != nil {
			return nil, err
		}
		typeMap[tid] = IMDBTitleType(titleType)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return typeMap, nil
}

var upsert_query_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TableName,
	strings.Join([]string{
		Column.TId,
		Column.Title,
		Column.OrigTitle,
		Column.Year,
		Column.Type,
		Column.IsAdult,
	}, ","),
)
var upsert_query_values_placeholder = "(" + util.RepeatJoin("?", 6, ",") + ")"
var upsert_query_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s`,
	Column.TId,
	Column.Title,
	Column.Title,
	Column.OrigTitle,
	Column.OrigTitle,
	Column.Year,
	Column.Year,
	Column.Type,
	Column.Type,
	Column.IsAdult,
	Column.IsAdult,
)

func get_upsert_query(count int) string {
	return upsert_query_before_values +
		util.RepeatJoin(upsert_query_values_placeholder, count, ",") +
		upsert_query_after_values
}

func Upsert(titles []IMDBTitle) error {
	if len(titles) == 0 {
		return nil
	}
	query := get_upsert_query(len(titles))
	var args []any
	for _, t := range titles {
		args = append(args, t.TId, t.Title, t.OrigTitle, t.Year, t.Type, t.IsAdult)
	}

	_, err := db.Exec(query, args...)
	return err
}

var rebuild_fts_query = fmt.Sprintf(
	`INSERT INTO %s_fts(%s_fts) VALUES('rebuild')`,
	TableName,
	TableName,
)

func sqliteRebuildFTS() error {
	_, err := db.Exec(rebuild_fts_query)
	return err
}

func postgresRebuildFTS() error {
	return nil
}

var RebuildFTS = func() func() error {
	if db.Dialect == db.DBDialectSQLite {
		return sqliteRebuildFTS
	}
	return postgresRebuildFTS
}()

var __sl_query_search_type_movie = fmt.Sprintf(
	`itf.%s IN ('%s', '%s')`,
	Column.Type,
	IMDBTitleTypeMovie,
	IMDBTitleTypeTvMovie,
)
var __sl_query_search_type_show = fmt.Sprintf(
	`itf.%s IN ('%s', '%s', '%s', '%s', '%s')`,
	Column.Type,
	IMDBTitleTypeShort,
	IMDBTitleTypeTvMiniSeries,
	IMDBTitleTypeTvSeries,
	IMDBTitleTypeTvShort,
	IMDBTitleTypeTvSpecial,
)
var __sl_query_search_year_eq = fmt.Sprintf(
	`itf.%s = ?`,
	Column.Year,
)
var __sl_query_search_year_between = fmt.Sprintf(
	`itf.%s BETWEEN ? AND ?`,
	Column.Year,
)
var sl_query_search_type_movie = " AND " + __sl_query_search_type_movie
var sl_query_search_type_show = " AND " + __sl_query_search_type_show
var sl_query_search_year_eq = " AND " + __sl_query_search_year_eq
var sl_query_search_year_between = " AND " + __sl_query_search_year_between

var sl_query_search_ids_select = fmt.Sprintf(
	"SELECT it.%s FROM %s_fts(?) itf JOIN %s it ON it.rowid = itf.rowid WHERE rank = 'bm25(10,10)'",
	Column.TId,
	TableName,
	TableName,
)
var sl_query_search_ids_order_by_limit = fmt.Sprintf(
	" ORDER BY CASE WHEN lower(itf.%s) = ? OR lower(itf.%s) = ? THEN 0 ELSE 1 END, rank LIMIT ?",
	Column.Title,
	Column.OrigTitle,
)

type SearchTitleType string

const (
	SearchTitleTypeMovie   SearchTitleType = "movie"
	SearchTitleTypeShow    SearchTitleType = "show"
	SearchTitleTypeUnknown SearchTitleType = ""
)

func sqliteSearchIds(title string, titleType SearchTitleType, year int, extendYear bool, limit int) ([]string, error) {
	title = strings.ToLower(title)

	fts_query := title
	if year != 0 && extendYear {
		fts_query += " " + strconv.Itoa(year)
	}
	fts_query = db.PrepareFTS5Query(fts_query, false)
	if fts_query == "" {
		return []string{}, nil
	}

	var query strings.Builder
	var args []any

	query.WriteString(sl_query_search_ids_select)
	args = append(args, fts_query)

	switch titleType {
	case SearchTitleTypeMovie:
		query.WriteString(sl_query_search_type_movie)
	case SearchTitleTypeShow:
		query.WriteString(sl_query_search_type_show)
	}

	if year != 0 {
		if extendYear {
			query.WriteString(sl_query_search_year_between)
			args = append(args, year-1, year+1)
		} else {
			query.WriteString(sl_query_search_year_eq)
			args = append(args, year)
		}
	}

	query.WriteString(sl_query_search_ids_order_by_limit)
	args = append(args, title, title)
	if limit == 0 {
		if year != 0 && titleType != "" {
			limit = 1
		} else if year != 0 || titleType != "" {
			limit = 3
		} else {
			limit = 5
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

	if len(ids) == 0 && titleType != "" {
		return sqliteSearchIds(title, "", year, extendYear, 0)
	}

	return ids, nil
}

var __pg_query_search_type_movie = fmt.Sprintf(
	`%s IN ('%s', '%s')`,
	Column.Type,
	IMDBTitleTypeMovie,
	IMDBTitleTypeTvMovie,
)
var __pg_query_search_type_show = fmt.Sprintf(
	`%s IN ('%s', '%s', '%s', '%s', '%s')`,
	Column.Type,
	IMDBTitleTypeShort,
	IMDBTitleTypeTvMiniSeries,
	IMDBTitleTypeTvSeries,
	IMDBTitleTypeTvShort,
	IMDBTitleTypeTvSpecial,
)
var __pg_query_search_year_eq = fmt.Sprintf(
	`%s = ?`,
	Column.Year,
)
var __pg_query_search_year_between = fmt.Sprintf(
	`%s BETWEEN ? AND ?`,
	Column.Year,
)
var pg_query_search_ids_select = fmt.Sprintf(
	"SELECT %s FROM %s WHERE search_vector @@ plainto_tsquery(?) ",
	Column.TId,
	TableName,
)
var pg_query_search_type_movie = " AND " + __pg_query_search_type_movie
var pg_query_search_type_show = " AND " + __pg_query_search_type_show
var pg_query_search_year_eq = " AND " + __pg_query_search_year_eq
var pg_query_search_year_between = " AND " + __pg_query_search_year_between
var pg_query_search_ids_order_by_limit = fmt.Sprintf(
	" ORDER BY CASE WHEN lower(%s) = ? OR lower(%s) = ? THEN 0 ELSE 1 END, -ts_rank(search_vector, plainto_tsquery(?)) LIMIT ?",
	Column.Title,
	Column.OrigTitle,
)

func postgresSearchIds(title string, titleType SearchTitleType, year int, extendYear bool, limit int) ([]string, error) {
	title = strings.ToLower(title)

	fts_query := title
	if year != 0 && extendYear {
		fts_query += " " + strconv.Itoa(year)
	}
	if fts_query == "" {
		return []string{}, nil
	}

	var query strings.Builder
	var args []any

	query.WriteString(pg_query_search_ids_select)
	args = append(args, fts_query)

	switch titleType {
	case SearchTitleTypeMovie:
		query.WriteString(pg_query_search_type_movie)
	case SearchTitleTypeShow:
		query.WriteString(pg_query_search_type_show)
	}

	if year != 0 {
		if extendYear {
			query.WriteString(pg_query_search_year_between)
			args = append(args, year-1, year+1)
		} else {
			query.WriteString(pg_query_search_year_eq)
			args = append(args, year)
		}
	}

	query.WriteString(pg_query_search_ids_order_by_limit)
	args = append(args, title, title, fts_query)

	if limit == 0 {
		if year != 0 && titleType != "" {
			limit = 1
		} else if year != 0 || titleType != "" {
			limit = 3
		} else {
			limit = 5
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

	if len(ids) == 0 && titleType != "" {
		return postgresSearchIds(title, "", year, extendYear, 0)
	}

	return ids, nil
}

var SearchIds = func() func(title string, titleType SearchTitleType, year int, extendYear bool, limit int) ([]string, error) {
	if db.Dialect == db.DBDialectSQLite {
		return sqliteSearchIds
	}
	return postgresSearchIds
}()

func sqliteSearchOne(title string, titleType SearchTitleType, year int, extendYear bool) (*IMDBTitle, error) {
	ids, err := sqliteSearchIds(title, titleType, year, extendYear, 0)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	items, err := ListByIds(ids)
	if err != nil {
		return nil, err
	}

	if len(items) == 1 {
		return &items[0], nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Year > items[j].Year
	})

	for i := range items {
		item := items[i]
		title_matched := strings.ToLower(item.Title) == title || strings.ToLower(item.OrigTitle) == title
		year_matched := year == 0 || item.Year == 0 || year == item.Year

		if title_matched && year_matched {
			return &item, nil
		}
	}

	return nil, nil
}

var pg_query_search_one_select = fmt.Sprintf(
	`SELECT %s FROM %s WHERE search_vector @@ plainto_tsquery(?)`,
	strings.Join([]string{
		Column.Id,
		Column.TId,
		Column.Title,
		Column.OrigTitle,
		Column.Year,
		Column.Type,
		Column.IsAdult,
	}, ","),
	TableName,
)

func postgresSearchOne(title string, titleType SearchTitleType, year int, extendYear bool) (*IMDBTitle, error) {
	title = strings.ToLower(title)

	var query strings.Builder
	var args []any

	query.WriteString(pg_query_search_one_select)

	fts_query := title
	args = append(args, fts_query)

	switch titleType {
	case SearchTitleTypeMovie:
		query.WriteString(pg_query_search_type_movie)
	case SearchTitleTypeShow:
		query.WriteString(pg_query_search_type_show)
	}

	if year != 0 {
		if extendYear {
			query.WriteString(pg_query_search_year_between)
			args = append(args, year-1, year+1)
		} else {
			query.WriteString(pg_query_search_year_eq)
			args = append(args, year)
		}
	}

	query.WriteString(pg_query_search_ids_order_by_limit)
	args = append(args, title, title, fts_query)

	if year != 0 && titleType != "" {
		args = append(args, 1)
	} else if year != 0 || titleType != "" {
		args = append(args, 3)
	} else {
		args = append(args, 5)
	}

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []IMDBTitle
	for rows.Next() {
		var title IMDBTitle
		if err := rows.Scan(
			&title.Id,
			&title.TId,
			&title.Title,
			&title.OrigTitle,
			&title.Year,
			&title.Type,
			&title.IsAdult,
		); err != nil {
			return nil, err
		}

		if title.OrigTitle == "" {
			title.OrigTitle = title.Title
		}

		titles = append(titles, title)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(titles) == 0 && titleType != "" {
		return postgresSearchOne(title, "", year, extendYear)
	}

	if len(titles) == 1 {
		return &titles[0], nil
	}

	sort.Slice(titles, func(i, j int) bool {
		return titles[i].Year > titles[j].Year
	})

	for i := range titles {
		item := titles[i]
		title_matched := strings.ToLower(item.Title) == title || strings.ToLower(item.OrigTitle) == title
		year_matched := year == 0 || item.Year == 0 || year == item.Year

		if title_matched && year_matched {
			return &item, nil
		}
	}

	return nil, nil
}

var SearchOne = func() func(title string, titleType SearchTitleType, year int, extendYear bool) (*IMDBTitle, error) {
	if db.Dialect == db.DBDialectSQLite {
		return sqliteSearchOne
	}
	return postgresSearchOne
}()
