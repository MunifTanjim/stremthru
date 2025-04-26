package imdb_title

import (
	"database/sql"
	"fmt"
	"regexp"
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
	Column.Id,
)

func ListByIds(ids []int) ([]IMDBTitle, error) {
	query := fmt.Sprintf("%s (%s)", query_by_ids_prefix, strings.Repeat("?,", len(ids)-1)+"?")
	args := make([]any, len(ids))
	for i, id := range ids {
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

var __query_search_type_movie = fmt.Sprintf(
	`%s IN ('%s', '%s')`,
	Column.Type,
	IMDBTitleTypeMovie,
	IMDBTitleTypeTvMovie,
)
var __query_search_type_show = fmt.Sprintf(
	`%s IN ('%s', '%s', '%s', '%s', '%s')`,
	Column.Type,
	IMDBTitleTypeShort,
	IMDBTitleTypeTvMiniSeries,
	IMDBTitleTypeTvSeries,
	IMDBTitleTypeTvShort,
	IMDBTitleTypeTvSpecial,
)
var __query_search_year_eq = fmt.Sprintf(
	`%s = ?`,
	Column.Year,
)
var __query_search_year_between = fmt.Sprintf(
	`%s BETWEEN ? AND ?`,
	Column.Year,
)
var query_search_type_movie = " AND " + __query_search_type_movie
var query_search_type_show = " AND " + __query_search_type_show
var query_search_year_eq = " AND " + __query_search_year_eq
var query_search_year_between = " AND " + __query_search_year_between
var query_search = fmt.Sprintf(
	`SELECT rowid, CASE WHEN lower(%s) = ? OR lower(%s) = ? THEN 0 ELSE 1 END AS priority FROM %s_fts(?) WHERE rank = 'bm25(10,10)'`,
	Column.Title,
	Column.OrigTitle,
	TableName,
)
var query_search_order_by_limit = ` ORDER BY priority, rank LIMIT ?`

type SearchTitleType string

const (
	SearchTitleTypeMovie   SearchTitleType = "movie"
	SearchTitleTypeShow    SearchTitleType = "show"
	SearchTitleTypeUnknown SearchTitleType = ""
)

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-z0-9]`)
var whitespacesRegex = regexp.MustCompile(`\s{2,}`)
var fts5SymbolRegex = regexp.MustCompile(`[-+*:^]`)

func fts5Query(query string) string {
	query = whitespacesRegex.ReplaceAllLiteralString(fts5SymbolRegex.ReplaceAllLiteralString(query, " "), " ")
	if strings.TrimSpace(query) == "" {
		return ""
	}
	return `"` + strings.Join(strings.Split(query, " "), `" "`) + `"`
}

func searchIds(title string, titleType SearchTitleType, year int, extendYear bool) ([]int, error) {
	title = strings.ToLower(title)

	fts_query := fts5Query(title)
	if fts_query == "" {
		return []int{}, nil
	}

	var query strings.Builder
	var args []any

	query.WriteString(query_search)

	args = append(args, title, title)
	args = append(args, fts_query)

	switch titleType {
	case SearchTitleTypeMovie:
		query.WriteString(query_search_type_movie)
	case SearchTitleTypeShow:
		query.WriteString(query_search_type_show)
	}

	if year != 0 {
		if extendYear {
			query.WriteString(query_search_year_between)
			args = append(args, year-1, year+1)
		} else {
			query.WriteString(query_search_year_eq)
			args = append(args, year)
		}
	}

	query.WriteString(query_search_order_by_limit)
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

	ids := []int{}
	if rows.Next() {
		var id int
		var priority int
		if err := rows.Scan(&id, &priority); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 && titleType != "" {
		return searchIds(title, "", year, extendYear)
	}

	return ids, nil
}

func sqliteSearchOne(title string, titleType SearchTitleType, year int, extendYear bool) (*IMDBTitle, error) {
	ids, err := searchIds(title, titleType, year, extendYear)
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

var pg_query_search_select = fmt.Sprintf(
	`SELECT %s, CASE WHEN lower(%s) = ? OR lower(%s) = ? THEN 0 ELSE 1 END AS priority, -ts_rank(search_vector, plainto_tsquery(?)) AS rank FROM %s WHERE search_vector @@ plainto_tsquery(?) `,
	strings.Join([]string{
		Column.Id,
		Column.TId,
		Column.Title,
		Column.OrigTitle,
		Column.Year,
		Column.Type,
		Column.IsAdult,
	}, ","),
	Column.Title,
	Column.OrigTitle,
	TableName,
)

func postgresSearchOne(title string, titleType SearchTitleType, year int, extendYear bool) (*IMDBTitle, error) {
	title = strings.ToLower(title)

	var query strings.Builder
	var args []any

	query.WriteString(pg_query_search_select)

	args = append(args, title, title)
	if year != 0 && extendYear {
		args = append(args, title+" "+strconv.Itoa(year), title+" "+strconv.Itoa(year))
	} else {
		args = append(args, title, title)
	}

	switch titleType {
	case SearchTitleTypeMovie:
		query.WriteString(query_search_type_movie)
	case SearchTitleTypeShow:
		query.WriteString(query_search_type_show)
	}

	if year != 0 {
		if extendYear {
			query.WriteString(query_search_year_between)
			args = append(args, year-1, year+1)
		} else {
			query.WriteString(query_search_year_eq)
			args = append(args, year)
		}
	}

	query.WriteString(query_search_order_by_limit)
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
		var priority int
		var rank float32
		if err := rows.Scan(
			&title.Id,
			&title.TId,
			&title.Title,
			&title.OrigTitle,
			&title.Year,
			&title.Type,
			&title.IsAdult,
			&priority,
			&rank,
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
