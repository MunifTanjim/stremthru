package anime

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const IdMapTableName = "anime_id_map"

type AnimeIdMapType = string

const (
	AnimeIdMapTypeTV      AnimeIdMapType = "TV"
	AnimeIdMapTypeTVShort AnimeIdMapType = "TV_SHORT"
	AnimeIdMapTypeMovie   AnimeIdMapType = "MOVIE"
	AnimeIdMapTypeSpecial AnimeIdMapType = "SPECIAL"
	AnimeIdMapTypeOVA     AnimeIdMapType = "OVA"
	AnimeIdMapTypeONA     AnimeIdMapType = "ONA"
	AnimeIdMapTypeMusic   AnimeIdMapType = "MUSIC"
	AnimeIdMapTypeManga   AnimeIdMapType = "MANGA"
	AnimeIdMapTypeNovel   AnimeIdMapType = "NOVEL"
	AnimeIdMapTypeOneShot AnimeIdMapType = "ONE_SHOT"
	AnimeIdMapTypeUnknown AnimeIdMapType = ""
)

type AnimeIdMap struct {
	Id          int            `json:"id"`
	Type        AnimeIdMapType `json:"type"`
	AniList     string         `json:"anilist"`
	AniDB       string         `json:"anidb"`
	AniSearch   string         `json:"anisearch"`
	AnimePlanet string         `json:"animeplanet"`
	IMDB        string         `json:"imdb"`
	Kitsu       string         `json:"kitsu"`
	LiveChart   string         `json:"livechart"`
	MAL         string         `json:"mal"`
	NotifyMoe   string         `json:"notifymoe"`
	TMDB        string         `json:"tmdb"`
	TVDB        string         `json:"tvdb"`
	UpdatedAt   db.Timestamp   `json:"uat"`
}

func (idMap *AnimeIdMap) IsZero() bool {
	return idMap.Id == 0
}

func (idMap *AnimeIdMap) IsStale() bool {
	return time.Now().After(idMap.UpdatedAt.Add(15 * 24 * time.Hour))
}

type rawAnimeIdMap struct {
	Id          int            `json:"id"`
	Type        AnimeIdMapType `json:"type"`
	AniList     db.NullString  `json:"anilist"`
	AniDB       db.NullString  `json:"anidb"`
	AniSearch   db.NullString  `json:"anisearch"`
	AnimePlanet db.NullString  `json:"animeplanet"`
	IMDB        db.NullString  `json:"imdb"`
	Kitsu       db.NullString  `json:"kitsu"`
	LiveChart   db.NullString  `json:"livechart"`
	MAL         db.NullString  `json:"mal"`
	NotifyMoe   db.NullString  `json:"notifymoe"`
	TMDB        db.NullString  `json:"tmdb"`
	TVDB        db.NullString  `json:"tvdb"`
	UpdatedAt   db.Timestamp   `json:"uat"`
}

type IdMapColumnStruct struct {
	Id          string
	Type        string
	AniDB       string
	AniList     string
	AniSearch   string
	AnimePlanet string
	IMDB        string
	Kitsu       string
	LiveChart   string
	MAL         string
	NotifyMoe   string
	TMDB        string
	TVDB        string
	UpdatedAt   string
}

var IdMapColumn = IdMapColumnStruct{
	Id:          "id",
	Type:        "type",
	AniDB:       "anidb",
	AniList:     "anilist",
	AniSearch:   "anisearch",
	AnimePlanet: "animeplanet",
	IMDB:        "imdb",
	Kitsu:       "kitsu",
	LiveChart:   "livechart",
	MAL:         "mal",
	NotifyMoe:   "notifymoe",
	TMDB:        "tmdb",
	TVDB:        "tvdb",
	UpdatedAt:   "uat",
}

var IdMapColumns = []string{
	IdMapColumn.Id,
	IdMapColumn.Type,
	IdMapColumn.AniDB,
	IdMapColumn.AniList,
	IdMapColumn.AniSearch,
	IdMapColumn.AnimePlanet,
	IdMapColumn.IMDB,
	IdMapColumn.Kitsu,
	IdMapColumn.LiveChart,
	IdMapColumn.MAL,
	IdMapColumn.NotifyMoe,
	IdMapColumn.TMDB,
	IdMapColumn.TVDB,
	IdMapColumn.UpdatedAt,
}

var query_get_id_map = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IN ",
	strings.Join(IdMapColumns, ","),
	IdMapTableName,
	IdMapColumn.AniList,
)

func GetIdMapsForAniList(ids []int) ([]AnimeIdMap, error) {
	count := len(ids)
	query := query_get_id_map + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i := range ids {
		args[i] = strconv.Itoa(ids[i])
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idMaps := []AnimeIdMap{}
	for rows.Next() {
		var item rawAnimeIdMap
		if err := rows.Scan(
			&item.Id,
			&item.Type,
			&item.AniDB,
			&item.AniList,
			&item.AniSearch,
			&item.AnimePlanet,
			&item.IMDB,
			&item.Kitsu,
			&item.LiveChart,
			&item.MAL,
			&item.NotifyMoe,
			&item.TMDB,
			&item.TVDB,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		idMaps = append(idMaps, AnimeIdMap{
			Id:          item.Id,
			Type:        item.Type,
			AniList:     item.AniList.String,
			AniDB:       item.AniDB.String,
			AniSearch:   item.AniSearch.String,
			AnimePlanet: item.AnimePlanet.String,
			IMDB:        item.IMDB.String,
			Kitsu:       item.Kitsu.String,
			LiveChart:   item.LiveChart.String,
			MAL:         item.MAL.String,
			NotifyMoe:   item.NotifyMoe.String,
			TMDB:        item.TMDB.String,
			TVDB:        item.TVDB.String,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return idMaps, nil
}

var query_get_type_by_anilist_ids = fmt.Sprintf(
	"SELECT %s, %s FROM %s WHERE %s IN ",
	IdMapColumn.AniList,
	IdMapColumn.Type,
	IdMapTableName,
	IdMapColumn.AniList,
)

func GetTypeByAnilistIds(ids []int) (map[int]AnimeIdMapType, error) {
	count := len(ids)
	if count == 0 {
		return nil, nil
	}

	query := query_get_type_by_anilist_ids + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i := range ids {
		args[i] = strconv.Itoa(ids[i])
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeById := make(map[int]AnimeIdMapType, count)
	for rows.Next() {
		var id string
		var animeType AnimeIdMapType
		if err := rows.Scan(&id, &animeType); err != nil {
			return nil, err
		}
		if id, err := strconv.Atoi(id); err == nil {
			typeById[id] = animeType
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return typeById, nil
}

var query_get_type_by_kitsu_ids = fmt.Sprintf(
	"SELECT %s, %s FROM %s WHERE %s IN ",
	IdMapColumn.Kitsu,
	IdMapColumn.Type,
	IdMapTableName,
	IdMapColumn.Kitsu,
)

func GetTypeByKitsuIds(ids []int) (map[int]AnimeIdMapType, error) {
	count := len(ids)
	if count == 0 {
		return nil, nil
	}

	query := query_get_type_by_kitsu_ids + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, count)
	for i := range ids {
		args[i] = strconv.Itoa(ids[i])
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeById := make(map[int]AnimeIdMapType, count)
	for rows.Next() {
		var id string
		var animeType AnimeIdMapType
		if err := rows.Scan(&id, &animeType); err != nil {
			return nil, err
		}
		if id, err := strconv.Atoi(id); err == nil {
			typeById[id] = animeType
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return typeById, nil
}

var query_bulk_record_id_maps_before_values = fmt.Sprintf(
	`INSERT INTO %s AS aim (%s) VALUES `,
	IdMapTableName,
	strings.Join(IdMapColumns[1:len(IdMapColumns)-1], ","),
)
var query_bulk_record_id_maps_placeholder = "(" + util.RepeatJoin("?", len(IdMapColumns)-2, ",") + ")"
var query_bulk_record_id_maps_on_conflict_before_column = " ON CONFLICT ("
var query_bulk_record_id_maps_on_conflict_after_column = fmt.Sprintf(
	`) DO UPDATE SET %s = CASE WHEN aim.%s = '' THEN EXCLUDED.%s ELSE aim.%s END, %s = %s`,
	IdMapColumn.Type,
	IdMapColumn.Type,
	IdMapColumn.Type,
	IdMapColumn.Type,
	IdMapColumn.UpdatedAt,
	db.CurrentTimestamp,
)
var query_bulk_record_id_maps_on_conflict_set_by_column = map[string]string{
	IdMapColumn.AniDB:       fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.AniDB, IdMapColumn.AniDB, IdMapColumn.AniDB, IdMapColumn.AniDB),
	IdMapColumn.AniList:     fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.AniList, IdMapColumn.AniList, IdMapColumn.AniList, IdMapColumn.AniList),
	IdMapColumn.AniSearch:   fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.AniSearch, IdMapColumn.AniSearch, IdMapColumn.AniSearch, IdMapColumn.AniSearch),
	IdMapColumn.AnimePlanet: fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.AnimePlanet, IdMapColumn.AnimePlanet, IdMapColumn.AnimePlanet, IdMapColumn.AnimePlanet),
	IdMapColumn.IMDB:        fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.IMDB, IdMapColumn.IMDB, IdMapColumn.IMDB, IdMapColumn.IMDB),
	IdMapColumn.Kitsu:       fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.Kitsu, IdMapColumn.Kitsu, IdMapColumn.Kitsu, IdMapColumn.Kitsu),
	IdMapColumn.LiveChart:   fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.LiveChart, IdMapColumn.LiveChart, IdMapColumn.LiveChart, IdMapColumn.LiveChart),
	IdMapColumn.MAL:         fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.MAL, IdMapColumn.MAL, IdMapColumn.MAL, IdMapColumn.MAL),
	IdMapColumn.NotifyMoe:   fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.NotifyMoe, IdMapColumn.NotifyMoe, IdMapColumn.NotifyMoe, IdMapColumn.NotifyMoe),
	IdMapColumn.TMDB:        fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.TMDB, IdMapColumn.TMDB, IdMapColumn.TMDB, IdMapColumn.TMDB),
	IdMapColumn.TVDB:        fmt.Sprintf("%s = CASE WHEN aim.%s IS NULL THEN EXCLUDED.%s ELSE aim.%s END", IdMapColumn.TVDB, IdMapColumn.TVDB, IdMapColumn.TVDB, IdMapColumn.TVDB),
}

func normalizeOptionalId(id string) string {
	if id == "0" {
		return ""
	}
	return id
}

func BulkRecordIdMaps(items []AnimeIdMap, anchorColumnName string) error {
	count := len(items)
	if count == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(query_bulk_record_id_maps_before_values)
	query.WriteString(util.RepeatJoin(query_bulk_record_id_maps_placeholder, count, ","))
	query.WriteString(query_bulk_record_id_maps_on_conflict_before_column)
	query.WriteString(anchorColumnName)
	query.WriteString(query_bulk_record_id_maps_on_conflict_after_column)
	for columnName, setColumnValue := range query_bulk_record_id_maps_on_conflict_set_by_column {
		if columnName == anchorColumnName {
			continue
		}
		query.WriteString(", ")
		query.WriteString(setColumnValue)
	}

	columnCount := len(IdMapColumns) - 2
	args := make([]any, count*columnCount)
	for i, item := range items {
		args[i*columnCount+0] = item.Type
		args[i*columnCount+1] = db.NullString{String: normalizeOptionalId(item.AniDB)}
		args[i*columnCount+2] = db.NullString{String: normalizeOptionalId(item.AniList)}
		args[i*columnCount+3] = db.NullString{String: normalizeOptionalId(item.AniSearch)}
		args[i*columnCount+4] = db.NullString{String: normalizeOptionalId(item.AnimePlanet)}
		args[i*columnCount+5] = db.NullString{String: normalizeOptionalId(item.IMDB)}
		args[i*columnCount+6] = db.NullString{String: normalizeOptionalId(item.Kitsu)}
		args[i*columnCount+7] = db.NullString{String: normalizeOptionalId(item.LiveChart)}
		args[i*columnCount+8] = db.NullString{String: normalizeOptionalId(item.MAL)}
		args[i*columnCount+9] = db.NullString{String: normalizeOptionalId(item.NotifyMoe)}
		args[i*columnCount+10] = db.NullString{String: normalizeOptionalId(item.TMDB)}
		args[i*columnCount+11] = db.NullString{String: normalizeOptionalId(item.TVDB)}
	}

	_, err := db.Exec(query.String(), args...)
	return err
}
