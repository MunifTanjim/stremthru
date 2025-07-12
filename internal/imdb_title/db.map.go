package imdb_title

import (
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const MapTableName = "imdb_title_map"

type IMDBTitleMap struct {
	IMDBId    string       `json:"imdb"`
	TMDBId    string       `json:"tmdb"`
	TVDBId    string       `json:"tvdb"`
	TraktId   string       `json:"trakt"`
	MALId     string       `json:"mal"`
	UpdatedAt db.Timestamp `json:"uat"`
}

type MapColumnStruct struct {
	IMDBId    string
	TMDBId    string
	TVDBId    string
	TraktId   string
	MALId     string
	UpdatedAt string
}

var MapColumn = MapColumnStruct{
	IMDBId:    "imdb",
	TMDBId:    "tmdb",
	TVDBId:    "tvdb",
	TraktId:   "trakt",
	MALId:     "mal",
	UpdatedAt: "uat",
}

var query_get_imdb_id_by_trakt_id = fmt.Sprintf(
	`SELECT it.%s, it.%s, itm.%s FROM %s itm JOIN %s it ON it.%s = itm.%s WHERE `,
	Column.TId,
	Column.Type,
	MapColumn.TraktId,
	MapTableName,
	TableName,
	Column.TId,
	MapColumn.IMDBId,
)
var query_get_imdb_id_by_trakt_id_cond_movie = fmt.Sprintf(
	` it.%s IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(movieTypes), ","),
		movieTypes[0],
		movieTypes[1],
	),
	MapColumn.TraktId,
)
var query_get_imdb_id_by_trakt_id_cond_show = fmt.Sprintf(
	` it.%s IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(showTypes), ","),
		showTypes[0],
		showTypes[1],
		showTypes[2],
		showTypes[3],
		showTypes[4],
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
		case movieTypes[0], movieTypes[1]:
			movieImdbIdByTraktId[traktId] = imdbId
		case showTypes[0], showTypes[1], showTypes[2], showTypes[3], showTypes[4]:
			showImdbIdByTraktId[traktId] = imdbId
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return movieImdbIdByTraktId, showImdbIdByTraktId, nil
}

var query_get_imdb_id_by_tmdb_id = fmt.Sprintf(
	`SELECT it.%s, it.%s, itm.%s FROM %s itm JOIN %s it ON it.%s = itm.%s WHERE `,
	Column.TId,
	Column.Type,
	MapColumn.TMDBId,
	MapTableName,
	TableName,
	Column.TId,
	MapColumn.IMDBId,
)
var query_get_imdb_id_by_tmdb_id_cond_movie = fmt.Sprintf(
	` it.%s IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(movieTypes), ","),
		movieTypes[0],
		movieTypes[1],
	),
	MapColumn.TMDBId,
)
var query_get_imdb_id_by_tmdb_id_cond_show = fmt.Sprintf(
	` it.%s IN (%s) AND itm.%s IN `,
	Column.Type,
	fmt.Sprintf(
		util.RepeatJoin("'%s'", len(showTypes), ","),
		showTypes[0],
		showTypes[1],
		showTypes[2],
		showTypes[3],
		showTypes[4],
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
		case movieTypes[0], movieTypes[1]:
			movieImdbIdByTMDBId[tmdbId] = imdbId
		case showTypes[0], showTypes[1], showTypes[2], showTypes[3], showTypes[4]:
			showImdbIdByTMDBId[tmdbId] = imdbId
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return movieImdbIdByTMDBId, showImdbIdByTMDBId, nil
}

func RecordMappingFromMDBList(tx *db.Tx, imdbId, tmdbId, tvdbId, traktId, malId string) error {
	query := fmt.Sprintf(
		`INSERT INTO %s AS itm (%s) VALUES (?,?,?,?,?) ON CONFLICT (%s) DO UPDATE SET %s, %s = %s`,
		MapTableName,
		db.JoinColumnNames(MapColumn.IMDBId, MapColumn.TMDBId, MapColumn.TVDBId, MapColumn.TraktId, MapColumn.MALId),
		MapColumn.IMDBId,
		strings.Join(
			[]string{
				fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId),
				fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId),
				fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId),
				fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.MALId, MapColumn.MALId, MapColumn.MALId, MapColumn.MALId),
			},
			", ",
		),
		MapColumn.UpdatedAt,
		db.CurrentTimestamp,
	)

	_, err := tx.Exec(query, imdbId, tmdbId, tvdbId, traktId, malId)
	return err
}

type BulkRecordMappingInputItem struct {
	IMDBId  string
	TMDBId  string
	TVDBId  string
	TraktId string
	MALId   string
}

var query_bulk_record_mapping_before_values = fmt.Sprintf(
	`INSERT INTO %s AS itm (%s,%s,%s,%s,%s) VALUES `,
	MapTableName,
	MapColumn.IMDBId,
	MapColumn.TMDBId,
	MapColumn.TVDBId,
	MapColumn.TraktId,
	MapColumn.MALId,
)
var query_bulk_record_mapping_placeholder = `(?,?,?,?,?)`
var query_bulk_record_mapping_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s, %s = %s`,
	MapColumn.IMDBId,
	strings.Join(
		[]string{
			fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId, MapColumn.TMDBId),
			fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId, MapColumn.TVDBId),
			fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId, MapColumn.TraktId),
			fmt.Sprintf("%s = CASE WHEN itm.%s = '' THEN EXCLUDED.%s ELSE itm.%s END", MapColumn.MALId, MapColumn.MALId, MapColumn.MALId, MapColumn.MALId),
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

func BulkRecordMapping(items []BulkRecordMappingInputItem) {
	count := len(items)
	query := query_bulk_record_mapping_before_values +
		util.RepeatJoin(query_bulk_record_mapping_placeholder, count, ",") +
		query_bulk_record_mapping_after_values

	args := make([]any, count*5)
	for i, item := range items {
		args[i*5+0] = item.IMDBId
		args[i*5+1] = normalizeOptionalId(item.TMDBId)
		args[i*5+2] = normalizeOptionalId(item.TVDBId)
		args[i*5+3] = normalizeOptionalId(item.TraktId)
		args[i*5+4] = normalizeOptionalId(item.MALId)
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		log.Error("failed to bulk record mapping", "error", err)
	}
}
