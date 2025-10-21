package imdb_title

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const MapTableName = "imdb_title_map"

type IMDBTitleMap struct {
	IMDBId       string       `json:"imdb"`
	TMDBId       string       `json:"tmdb"`
	TVDBId       string       `json:"tvdb"`
	TraktId      string       `json:"trakt"`
	LetterboxdId string       `json:"lboxd"`
	MALId        string       `json:"mal"`
	UpdatedAt    db.Timestamp `json:"uat"`

	Type IMDBTitleType `json:"-"`
}

type MapColumnStruct struct {
	IMDBId       string
	TMDBId       string
	TVDBId       string
	TraktId      string
	LetterboxdId string
	MALId        string
	UpdatedAt    string
}

var MapColumn = MapColumnStruct{
	IMDBId:       "imdb",
	TMDBId:       "tmdb",
	TVDBId:       "tvdb",
	TraktId:      "trakt",
	LetterboxdId: "lboxd",
	MALId:        "mal",
	UpdatedAt:    "uat",
}

var query_get_imdb_id_by_letterboxd_id = fmt.Sprintf(
	`SELECT %s, %s FROM %s WHERE %s IN `,
	MapColumn.IMDBId,
	MapColumn.LetterboxdId,
	MapTableName,
	MapColumn.LetterboxdId,
)

func GetIMDBIdByLetterboxdId(letterboxdIds []string) (map[string]string, error) {
	count := len(letterboxdIds)
	if count == 0 {
		return nil, nil
	}

	query := query_get_imdb_id_by_letterboxd_id + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i, id := range letterboxdIds {
		args[i] = id
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	imdbIdByLetterboxdId := make(map[string]string, count)
	for rows.Next() {
		var imdbId, letterboxdId string
		if err := rows.Scan(&imdbId, &letterboxdId); err != nil {
			return nil, err
		}
		imdbIdByLetterboxdId[letterboxdId] = imdbId
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return imdbIdByLetterboxdId, nil
}

var query_get_imdb_id_by_trakt_id = fmt.Sprintf(
	`SELECT itm.%s, coalesce(it.%s, '') AS item_type, itm.%s FROM %s itm LEFT JOIN %s it ON it.%s = itm.%s WHERE `,
	MapColumn.IMDBId,
	Column.Type,
	MapColumn.TraktId,
	MapTableName,
	TableName,
	Column.TId,
	MapColumn.IMDBId,
)
var query_get_imdb_id_by_trakt_id_cond_movie = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s,'') AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(movieTypes), ","),
		movieTypes[0],
		movieTypes[1],
		movieTypes[2],
	),
	MapColumn.TraktId,
)
var query_get_imdb_id_by_trakt_id_cond_show = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(showTypes), ","),
		showTypes[0],
		showTypes[1],
		showTypes[2],
		showTypes[3],
	),
	MapColumn.TraktId,
)

func GetIMDBIdByTraktId(traktMovieIds, traktShowIds []string) (map[string]string, map[string]string, error) {
	movieCount := len(traktMovieIds)
	showCount := len(traktShowIds)
	if movieCount+showCount == 0 {
		return nil, nil, nil
	}

	args := make([]any, movieCount+showCount)
	var query strings.Builder
	query.WriteString(query_get_imdb_id_by_trakt_id)
	if movieCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_trakt_id_cond_movie)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", movieCount, ","))
		query.WriteString("))")
		for i := range traktMovieIds {
			args[i] = traktMovieIds[i]
		}
		if showCount > 0 {
			query.WriteString(" OR ")
		}
	}
	if showCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_trakt_id_cond_show)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", showCount, ","))
		query.WriteString("))")
		for i := range traktShowIds {
			args[movieCount+i] = traktShowIds[i]
		}
	}

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	movieImdbIdByTraktId := make(map[string]string, movieCount)
	showImdbIdByTraktId := make(map[string]string, showCount)
	for rows.Next() {
		var imdbId string
		var imdbType IMDBTitleType
		var traktId string
		if err := rows.Scan(&imdbId, &imdbType, &traktId); err != nil {
			return nil, nil, err
		}
		switch imdbType {
		case movieTypes[0], movieTypes[1], movieTypes[2]:
			movieImdbIdByTraktId[traktId] = imdbId
		case showTypes[0], showTypes[1], showTypes[2], showTypes[3]:
			showImdbIdByTraktId[traktId] = imdbId
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return movieImdbIdByTraktId, showImdbIdByTraktId, nil
}

var query_get_imdb_id_by_tmdb_id = fmt.Sprintf(
	`SELECT itm.%s, coalesce(it.%s, '') AS item_type, itm.%s FROM %s itm LEFT JOIN %s it ON it.%s = itm.%s WHERE `,
	MapColumn.IMDBId,
	Column.Type,
	MapColumn.TMDBId,
	MapTableName,
	TableName,
	Column.TId,
	MapColumn.IMDBId,
)
var query_get_imdb_id_by_tmdb_id_cond_movie = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s,'') AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(movieTypes), ","),
		movieTypes[0],
		movieTypes[1],
		movieTypes[2],
	),
	MapColumn.TMDBId,
)
var query_get_imdb_id_by_tmdb_id_cond_show = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(showTypes), ","),
		showTypes[0],
		showTypes[1],
		showTypes[2],
		showTypes[3],
	),
	MapColumn.TMDBId,
)

func GetIMDBIdByTMDBId(tmdbMovieIds, tmdbShowIds []string) (map[string]string, map[string]string, error) {
	movieCount := len(tmdbMovieIds)
	showCount := len(tmdbShowIds)
	if movieCount+showCount == 0 {
		return nil, nil, nil
	}

	args := make([]any, movieCount+showCount)
	var query strings.Builder
	query.WriteString(query_get_imdb_id_by_tmdb_id)
	if movieCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_tmdb_id_cond_movie)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", movieCount, ","))
		query.WriteString("))")
		for i := range tmdbMovieIds {
			args[i] = tmdbMovieIds[i]
		}
		if showCount > 0 {
			query.WriteString(" OR ")
		}
	}
	if showCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_tmdb_id_cond_show)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", showCount, ","))
		query.WriteString("))")
		for i := range tmdbShowIds {
			args[movieCount+i] = tmdbShowIds[i]
		}
	}

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	movieImdbIdByTMDBId := make(map[string]string, movieCount)
	showImdbIdByTMDBId := make(map[string]string, showCount)
	for rows.Next() {
		var imdbId string
		var imdbType IMDBTitleType
		var tmdbId string
		if err := rows.Scan(&imdbId, &imdbType, &tmdbId); err != nil {
			return nil, nil, err
		}
		switch imdbType {
		case movieTypes[0], movieTypes[1], movieTypes[2]:
			movieImdbIdByTMDBId[tmdbId] = imdbId
		case showTypes[0], showTypes[1], showTypes[2], showTypes[3]:
			showImdbIdByTMDBId[tmdbId] = imdbId
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return movieImdbIdByTMDBId, showImdbIdByTMDBId, nil
}

var query_get_imdb_id_by_tvdb_id = fmt.Sprintf(
	`SELECT itm.%s, coalesce(it.%s, '') AS item_type, itm.%s FROM %s itm LEFT JOIN %s it ON it.%s = itm.%s WHERE `,
	MapColumn.IMDBId,
	Column.Type,
	MapColumn.TVDBId,
	MapTableName,
	TableName,
	Column.TId,
	MapColumn.IMDBId,
)
var query_get_imdb_id_by_tvdb_id_cond_movie = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s,'') AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(movieTypes), ","),
		movieTypes[0],
		movieTypes[1],
		movieTypes[2],
	),
	MapColumn.TVDBId,
)
var query_get_imdb_id_by_tvdb_id_cond_show = fmt.Sprintf(
	` coalesce(it.%s, '') IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(showTypes), ","),
		showTypes[0],
		showTypes[1],
		showTypes[2],
		showTypes[3],
	),
	MapColumn.TVDBId,
)

func GetIMDBIdByTVDBId(tvdbMovieIds, tvdbShowIds []string) (map[string]string, map[string]string, error) {
	movieCount := len(tvdbMovieIds)
	showCount := len(tvdbShowIds)
	if movieCount+showCount == 0 {
		return nil, nil, nil
	}

	args := make([]any, movieCount+showCount)
	var query strings.Builder
	query.WriteString(query_get_imdb_id_by_tvdb_id)
	if movieCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_tvdb_id_cond_movie)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", movieCount, ","))
		query.WriteString("))")
		for i := range tvdbMovieIds {
			args[i] = tvdbMovieIds[i]
		}
		if showCount > 0 {
			query.WriteString(" OR ")
		}
	}
	if showCount > 0 {
		query.WriteString("(")
		query.WriteString(query_get_imdb_id_by_tvdb_id_cond_show)
		query.WriteString("(")
		query.WriteString(util.RepeatJoin("?", showCount, ","))
		query.WriteString("))")
		for i := range tvdbShowIds {
			args[movieCount+i] = tvdbShowIds[i]
		}
	}

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	movieImdbIdByTVDBId := make(map[string]string, movieCount)
	showImdbIdByTVDBId := make(map[string]string, showCount)
	for rows.Next() {
		var imdbId string
		var imdbType IMDBTitleType
		var tvdbId string
		if err := rows.Scan(&imdbId, &imdbType, &tvdbId); err != nil {
			return nil, nil, err
		}
		switch imdbType.ToSimple() {
		case IMDBTitleSimpleTypeMovie:
			movieImdbIdByTVDBId[tvdbId] = imdbId
		case IMDBTitleSimpleTypeShow:
			showImdbIdByTVDBId[tvdbId] = imdbId
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return movieImdbIdByTVDBId, showImdbIdByTVDBId, nil
}

var query_get_tmdb_id_by_imdb_id = fmt.Sprintf(
	`SELECT %s, %s FROM %s WHERE %s IN `,
	MapColumn.IMDBId,
	MapColumn.TMDBId,
	MapTableName,
	MapColumn.IMDBId,
)

func GetTMDBIdByIMDBId(imdbIds []string) (map[string]string, error) {
	count := len(imdbIds)
	if count == 0 {
		return nil, nil
	}

	query := query_get_tmdb_id_by_imdb_id + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i, id := range imdbIds {
		args[i] = id
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tmdbIdByImdbId := make(map[string]string, count)
	for rows.Next() {
		var imdbId, tmdbId string
		if err := rows.Scan(&imdbId, &tmdbId); err != nil {
			return nil, err
		}
		tmdbIdByImdbId[imdbId] = tmdbId
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tmdbIdByImdbId, nil
}

var query_get_tvdb_id_by_imdb_id = fmt.Sprintf(
	`SELECT %s, %s FROM %s WHERE %s IN `,
	MapColumn.IMDBId,
	MapColumn.TVDBId,
	MapTableName,
	MapColumn.IMDBId,
)

func GetTVDBIdByIMDBId(imdbIds []string) (map[string]string, error) {
	count := len(imdbIds)
	if count == 0 {
		return nil, nil
	}

	query := query_get_tvdb_id_by_imdb_id + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i, id := range imdbIds {
		args[i] = id
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tvdbIdByImdbId := make(map[string]string, count)
	for rows.Next() {
		var imdbId, tvdbId string
		if err := rows.Scan(&imdbId, &tvdbId); err != nil {
			return nil, err
		}
		tvdbIdByImdbId[imdbId] = tvdbId
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tvdbIdByImdbId, nil
}

func RecordMappingFromMDBList(tx *db.Tx, imdbId, tmdbId, tvdbId, traktId, malId string) error {
	query := fmt.Sprintf(
		`INSERT INTO %s AS itm (%s) VALUES (?,?,?,?,?) ON CONFLICT (%s) DO UPDATE SET %s, %s = %s`,
		MapTableName,
		db.JoinColumnNames(MapColumn.IMDBId, MapColumn.TMDBId, MapColumn.TVDBId, MapColumn.TraktId, MapColumn.MALId),
		MapColumn.IMDBId,
		strings.Join(
			[]string{
				fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId),
				fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId),
				fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId),
				fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.MALId, MapColumn.MALId, MapColumn.MALId, MapColumn.MALId),
			},
			", ",
		),
		MapColumn.UpdatedAt,
		db.CurrentTimestamp,
	)

	_, err := tx.Exec(
		query,
		imdbId,
		normalizeOptionalId(tmdbId),
		normalizeOptionalId(tvdbId),
		normalizeOptionalId(traktId),
		normalizeOptionalId(malId),
	)
	return err
}

type BulkRecordMappingInputItem struct {
	IMDBId       string
	TMDBId       string
	TVDBId       string
	TraktId      string
	LetterboxdId string
	MALId        string
}

var query_bulk_record_mapping_before_values = fmt.Sprintf(
	`INSERT INTO %s AS itm (%s,%s,%s,%s,%s,%s) VALUES `,
	MapTableName,
	MapColumn.IMDBId,
	MapColumn.TMDBId,
	MapColumn.TVDBId,
	MapColumn.TraktId,
	MapColumn.LetterboxdId,
	MapColumn.MALId,
)
var query_bulk_record_mapping_placeholder = `(?,?,?,?,?,?)`
var query_bulk_record_mapping_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s, %s = %s`,
	MapColumn.IMDBId,
	strings.Join(
		[]string{
			fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId),
			fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId),
			fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId),
			fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.LetterboxdId, MapColumn.LetterboxdId, MapColumn.LetterboxdId, MapColumn.LetterboxdId),
			fmt.Sprintf("%s = CASE WHEN itm.%s IN ('','0') THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.MALId, MapColumn.MALId, MapColumn.MALId, MapColumn.MALId),
		},
		", ",
	),
	MapColumn.UpdatedAt,
	db.CurrentTimestamp,
)

func normalizeOptionalId(id string) string {
	if id == "0" {
		return ""
	}
	return id
}

func BulkRecordMapping(tx db.Executor, items []BulkRecordMappingInputItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}

	query := query_bulk_record_mapping_before_values +
		util.RepeatJoin(query_bulk_record_mapping_placeholder, count, ",") +
		query_bulk_record_mapping_after_values

	args := make([]any, count*6)
	for i, item := range items {
		args[i*6+0] = item.IMDBId
		args[i*6+1] = normalizeOptionalId(item.TMDBId)
		args[i*6+2] = normalizeOptionalId(item.TVDBId)
		args[i*6+3] = normalizeOptionalId(item.TraktId)
		args[i*6+4] = normalizeOptionalId(item.LetterboxdId)
		args[i*6+5] = normalizeOptionalId(item.MALId)
	}

	_, err := tx.Exec(query, args...)
	return err
}

var query_get_id_map_by_imdb_id = fmt.Sprintf(
	`SELECT %s, it.%s FROM %s itm LEFT JOIN %s it ON itm.%s = it.%s WHERE itm.%s = ?`,
	db.JoinPrefixedColumnNames(
		"itm.",
		MapColumn.IMDBId,
		MapColumn.TMDBId,
		MapColumn.TVDBId,
		MapColumn.TraktId,
		MapColumn.LetterboxdId,
		MapColumn.MALId,
	),
	Column.Type,
	MapTableName,
	TableName,
	MapColumn.IMDBId,
	Column.TId,
	MapColumn.IMDBId,
)

func GetIdMapByIMDBId(imdbId string) (*IMDBTitleMap, error) {
	var idMap IMDBTitleMap
	err := db.QueryRow(query_get_id_map_by_imdb_id, imdbId).Scan(
		&idMap.IMDBId,
		&idMap.TMDBId,
		&idMap.TVDBId,
		&idMap.TraktId,
		&idMap.LetterboxdId,
		&idMap.MALId,
		&idMap.Type,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &idMap, nil
}

var query_get_id_map_by_tvdb_id = fmt.Sprintf(
	`SELECT %s, it.%s FROM %s itm LEFT JOIN %s it ON itm.%s = it.%s WHERE itm.%s = ?`,
	db.JoinPrefixedColumnNames(
		"itm.",
		MapColumn.IMDBId,
		MapColumn.TMDBId,
		MapColumn.TVDBId,
		MapColumn.TraktId,
		MapColumn.LetterboxdId,
		MapColumn.MALId,
	),
	Column.Type,
	MapTableName,
	TableName,
	MapColumn.IMDBId,
	Column.TId,
	MapColumn.TVDBId,
)

func GetIdMapByTVDBId(tvdbId string) (*IMDBTitleMap, error) {
	var idMap IMDBTitleMap
	err := db.QueryRow(query_get_id_map_by_tvdb_id, tvdbId).Scan(
		&idMap.IMDBId,
		&idMap.TMDBId,
		&idMap.TVDBId,
		&idMap.TraktId,
		&idMap.LetterboxdId,
		&idMap.MALId,
		&idMap.Type,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &idMap, nil
}

var query_get_id_maps_by_letterboxd_id = fmt.Sprintf(
	`SELECT %s, coalesce(it.%s, '') AS item_type FROM %s itm LEFT JOIN %s it ON itm.%s = it.%s WHERE itm.%s IN `,
	db.JoinPrefixedColumnNames(
		"itm.",
		MapColumn.IMDBId,
		MapColumn.TMDBId,
		MapColumn.TVDBId,
		MapColumn.TraktId,
		MapColumn.LetterboxdId,
		MapColumn.MALId,
	),
	Column.Type,
	MapTableName,
	TableName,
	MapColumn.IMDBId,
	Column.TId,
	MapColumn.LetterboxdId,
)

func GetIdMapsByLetterboxdId(letterboxdIds []string) (map[string]IMDBTitleMap, error) {
	count := len(letterboxdIds)
	if count == 0 {
		return nil, nil
	}

	query := query_get_id_maps_by_letterboxd_id + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i, id := range letterboxdIds {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idMapById := make(map[string]IMDBTitleMap, count)
	for rows.Next() {
		idMap := IMDBTitleMap{}
		if err := rows.Scan(
			&idMap.IMDBId,
			&idMap.TMDBId,
			&idMap.TVDBId,
			&idMap.TraktId,
			&idMap.LetterboxdId,
			&idMap.MALId,
			&idMap.Type,
		); err != nil {
			return nil, err
		}

		idMapById[idMap.LetterboxdId] = idMap
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return idMapById, nil
}
