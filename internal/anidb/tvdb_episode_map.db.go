package anidb

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TVDBEpisodeMapTableName = "anidb_tvdb_episode_map"

// ;<anidb-special-episode>-<anidb-regular-episode>;...;
type AniDBTVDBEpisodeMapBefore map[int]int

func (before AniDBTVDBEpisodeMapBefore) Value() (driver.Value, error) {
	if len(before) == 0 {
		return "", nil
	}
	var value strings.Builder
	value.WriteRune(';')
	for special, regular := range before {
		value.WriteString(strconv.Itoa(special))
		value.WriteRune('-')
		value.WriteString(strconv.Itoa(regular))
		value.WriteRune(';')
	}
	return value.String(), nil
}

func (before AniDBTVDBEpisodeMapBefore) Scan(value any) error {
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New("failed to convert value to string")
	}
	for part := range strings.FieldsFuncSeq(strings.Trim(str, ";"), func(c rune) bool {
		return c == ';'
	}) {
		if specialStr, regularStr, ok := strings.Cut(part, "-"); ok {
			special, sErr := strconv.Atoi(specialStr)
			if sErr != nil {
				continue
			}
			regular, rErr := strconv.Atoi(regularStr)
			if rErr != nil {
				continue
			}
			before[special] = regular
		}
	}
	return nil
}

// ;<anidb-episode>-<tvdb-episode>[+<tvdb-episode>...];...;
type AniDBTVDBEpisodeMapMap map[int][]int

func (m AniDBTVDBEpisodeMapMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "", nil
	}
	var value strings.Builder
	value.WriteRune(';')
	for anidb, tvdb := range m {
		value.WriteString(strconv.Itoa(anidb))
		value.WriteRune('-')
		value.WriteString(strings.Join(util.SliceMapIntToString(tvdb), "+"))
		value.WriteRune(';')
	}
	return value.String(), nil
}

func (m AniDBTVDBEpisodeMapMap) Scan(value any) error {
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New("failed to convert value to string")
	}
	for part := range strings.FieldsFuncSeq(strings.Trim(str, ";"), func(c rune) bool {
		return c == ';'
	}) {
		if anidbStr, tvdbStr, ok := strings.Cut(part, "-"); ok {
			anidb, sErr := strconv.Atoi(anidbStr)
			if sErr != nil {
				continue
			}
			for str := range strings.SplitSeq(tvdbStr, "+") {
				tvdb, rErr := strconv.Atoi(str)
				if rErr != nil {
					continue
				}
				if m[anidb] == nil {
					m[anidb] = []int{}
				}
				m[anidb] = append(m[anidb], tvdb)
			}
		}
	}
	return nil
}

type AniDBTVDBEpisodeMap struct {
	AniDBId     string
	TVDBId      string
	AniDBSeason int // 0 for special, 1 for regular
	TVDBSeason  int // -1 for absolute order, 0 for specials
	Start       int
	End         int
	Offset      int
	Before      AniDBTVDBEpisodeMapBefore
	Map         AniDBTVDBEpisodeMapMap
}

func (m AniDBTVDBEpisodeMap) HasAbsoluteOrder() bool {
	return m.TVDBSeason == -1
}

func (m AniDBTVDBEpisodeMap) getTMDBEpisodes(anidbEpisode int) []int {
	if tvdbEpisode, ok := m.Map[anidbEpisode]; ok {
		return tvdbEpisode
	}
	if m.HasAbsoluteOrder() {
		return []int{anidbEpisode + m.Offset}
	}
	if m.End != 0 && anidbEpisode > m.End {
		return nil
	}
	if m.Start != 0 && anidbEpisode < m.Start {
		return nil
	}
	return []int{anidbEpisode + m.Offset}
}

func (m AniDBTVDBEpisodeMap) GetTMDBEpisode(anidbEpisode int) int {
	tvdbEpisodes := m.getTMDBEpisodes(anidbEpisode)
	if len(tvdbEpisodes) == 0 {
		return 0
	}
	return tvdbEpisodes[0]
}

var TVDBEpisodeMapColumn = struct {
	AniDBId     string
	TVDBId      string
	AniDBSeason string
	TVDBSeason  string
	Start       string
	End         string
	Offset      string
	Before      string
	Map         string
}{
	AniDBId:     "anidb_id",
	TVDBId:      "tvdb_id",
	AniDBSeason: "anidb_season",
	TVDBSeason:  "tvdb_season",
	Start:       "start",
	End:         "end",
	Offset:      "offset",
	Before:      "before",
	Map:         "map",
}

var TVDBEpisodeMapColumns = []string{
	TVDBEpisodeMapColumn.AniDBId,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapColumn.AniDBSeason,
	TVDBEpisodeMapColumn.TVDBSeason,
	TVDBEpisodeMapColumn.Start,
	TVDBEpisodeMapColumn.End,
	TVDBEpisodeMapColumn.Offset,
	TVDBEpisodeMapColumn.Before,
	TVDBEpisodeMapColumn.Map,
}

var query_upsert_episode_maps_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TVDBEpisodeMapTableName,
	strings.Join(TVDBEpisodeMapColumns, ","),
)
var query_upsert_episode_maps_values_placeholder = "(" + util.RepeatJoin("?", len(TVDBEpisodeMapColumns), ",") + ")"
var query_upsert_episode_maps_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s,%s,%s) DO UPDATE SET %s`,
	TVDBEpisodeMapColumn.AniDBId,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapColumn.AniDBSeason,
	TVDBEpisodeMapColumn.TVDBSeason,
	strings.Join([]string{
		fmt.Sprintf("%s = EXCLUDED.%s", TVDBEpisodeMapColumn.Start, TVDBEpisodeMapColumn.Start),
		fmt.Sprintf("%s = EXCLUDED.%s", TVDBEpisodeMapColumn.End, TVDBEpisodeMapColumn.End),
		fmt.Sprintf("%s = EXCLUDED.%s", TVDBEpisodeMapColumn.Offset, TVDBEpisodeMapColumn.Offset),
		fmt.Sprintf("%s = EXCLUDED.%s", TVDBEpisodeMapColumn.Before, TVDBEpisodeMapColumn.Before),
		fmt.Sprintf("%s = EXCLUDED.%s", TVDBEpisodeMapColumn.Map, TVDBEpisodeMapColumn.Map),
	}, ", "),
)

func UpsertTVDBEpisodeMaps(items []AniDBTVDBEpisodeMap) error {
	if len(items) == 0 {
		return nil
	}
	for cItems := range slices.Chunk(items, 200) {
		count := len(cItems)
		args := make([]any, count*len(TVDBEpisodeMapColumns))
		for i, item := range cItems {
			args[i*9+0] = item.AniDBId
			args[i*9+1] = item.TVDBId
			args[i*9+2] = item.AniDBSeason
			args[i*9+3] = item.TVDBSeason
			args[i*9+4] = item.Start
			args[i*9+5] = item.End
			args[i*9+6] = item.Offset
			args[i*9+7] = item.Before
			args[i*9+8] = item.Map
		}
		query := query_upsert_episode_maps_before_values + util.RepeatJoin(query_upsert_episode_maps_values_placeholder, count, ",") + query_upsert_episode_maps_after_values
		_, err := db.Exec(query, args...)
		return err
	}
	return nil
}

type AniDBTVDBEpisodeMaps []AniDBTVDBEpisodeMap

func (ms AniDBTVDBEpisodeMaps) GetTVDBId() string {
	return ms[0].TVDBId
}

func (ms AniDBTVDBEpisodeMaps) GetAniDBTitles() ([]AniDBTitle, error) {
	anidbIds := []string{}
	for i := range ms {
		anidbIds = append(anidbIds, ms[i].AniDBId)
	}
	return GetTitlesByIds(anidbIds)
}

func (ms AniDBTVDBEpisodeMaps) HasAbsoluteOrder() bool {
	for i := range ms {
		if ms[i].HasAbsoluteOrder() {
			return true
		}
	}
	return false
}

var query_get_tvdb_episode_maps_by_anidbid = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = (SELECT %s FROM %s WHERE %s = ?)`,
	strings.Join(TVDBEpisodeMapColumns, ","),
	TVDBEpisodeMapTableName,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapTableName,
	TVDBEpisodeMapColumn.AniDBId,
)

func GetTVDBEpisodeMaps(anidbId string) (AniDBTVDBEpisodeMaps, error) {
	rows, err := db.Query(query_get_tvdb_episode_maps_by_anidbid, anidbId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maps := []AniDBTVDBEpisodeMap{}
	for rows.Next() {
		var m AniDBTVDBEpisodeMap
		if err := rows.Scan(
			&m.AniDBId,
			&m.TVDBId,
			&m.AniDBSeason,
			&m.TVDBSeason,
			&m.Start,
			&m.End,
			&m.Offset,
			&m.Before,
			&m.Map,
		); err != nil {
			return nil, err
		}
		maps = append(maps, m)
	}
	return maps, nil
}
