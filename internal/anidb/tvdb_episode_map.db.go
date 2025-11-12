package anidb

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

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

func (m AniDBTVDBEpisodeMap) IsAniDBSpecialSeason() bool {
	return m.AniDBSeason == 0
}

func (m AniDBTVDBEpisodeMap) IsAniDBRegularSeason() bool {
	return m.AniDBSeason == 1
}

func (m AniDBTVDBEpisodeMap) HasEpisodeRange() bool {
	return m.Start > 0 && m.End > 0
}

func (m AniDBTVDBEpisodeMap) AniDBEpisodeBoundary() (int, int) {
	return m.Start, m.End
}

func (m AniDBTVDBEpisodeMap) TVDBEpisodeStart() int {
	return m.Start + m.Offset
}

func (m AniDBTVDBEpisodeMap) TVDBEpisodeEnd() int {
	return m.End + m.Offset
}

func (m AniDBTVDBEpisodeMap) TVDBEpisodeBoundary() (int, int) {
	return m.TVDBEpisodeStart(), m.TVDBEpisodeEnd()
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
	if anidbEpisode == -1 {
		return anidbEpisode
	}
	tvdbEpisodes := m.getTMDBEpisodes(anidbEpisode)
	if len(tvdbEpisodes) == 0 {
		return -1
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
	db.JoinColumnNames(TVDBEpisodeMapColumns...),
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
		fmt.Sprintf(`"%s" = EXCLUDED."%s"`, TVDBEpisodeMapColumn.End, TVDBEpisodeMapColumn.End),
		fmt.Sprintf(`"%s" = EXCLUDED."%s"`, TVDBEpisodeMapColumn.Offset, TVDBEpisodeMapColumn.Offset),
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

func (ms AniDBTVDBEpisodeMaps) Sort() {
	slices.SortFunc(ms, func(a, b AniDBTVDBEpisodeMap) int {
		aAniDBId, bAniDBId := util.SafeParseInt(a.AniDBId, -1), util.SafeParseInt(b.AniDBId, -1)
		if aAniDBId != bAniDBId {
			return aAniDBId - bAniDBId
		}
		if a.TVDBSeason != b.TVDBSeason {
			return a.TVDBSeason - b.TVDBSeason
		}
		if a.AniDBSeason != b.AniDBSeason {
			return a.AniDBSeason - b.AniDBSeason
		}
		if a.Start != b.Start {
			return a.Start - b.Start
		}
		return 0
	})
}

func (ms AniDBTVDBEpisodeMaps) GetByAnidbEpisode(ep int) *AniDBTVDBEpisodeMap {
	if ep == -1 {
		return nil
	}
	for i := range ms {
		m := &ms[i]
		if m.TVDBSeason <= 0 {
			continue
		}
		switch {
		case m.HasEpisodeRange():
			if ep >= m.Start && ep <= m.End {
				return m
			}
		case m.Start > 0:
			if m.Start <= ep {
				return m
			}
		case m.End > 0:
			if ep <= m.End {
				return m
			}
		}
	}
	return nil
}

func (ms AniDBTVDBEpisodeMaps) GetTVDBId() string {
	return ms[0].TVDBId
}

func (ms AniDBTVDBEpisodeMaps) GetTVDBSeasons() []int {
	seasons := []int{}
	for i := range ms {
		if !slices.Contains(seasons, ms[i].TVDBSeason) {
			seasons = append(seasons, ms[i].TVDBSeason)
		}
	}
	return seasons
}

func (ms AniDBTVDBEpisodeMaps) GetTVDBAbsoluteEpisode(tvSeason int, tvEpisode int) int {
	for i := range ms {
		m := &ms[i]
		if m.TVDBSeason == tvSeason {
			return tvEpisode - m.Offset + ms.GetAbsoluteOrderSeasonMap().Offset
		}
	}
	return -1
}

func (ms AniDBTVDBEpisodeMaps) ToTVDBAbsoluteRange(tvSeason int, tvEpisodeStart, tvEpisodeEnd int) (int, int) {
	for i := range ms {
		m := &ms[i]
		if m.TVDBSeason == tvSeason {
			absOffset := ms.GetAbsoluteOrderSeasonMap().Offset
			start := tvEpisodeStart - m.Offset + absOffset
			return start, tvEpisodeEnd + (start - tvEpisodeEnd)
		}
	}
	return -1, -1
}

// multiple tv seasons for the same anime season
func (ms AniDBTVDBEpisodeMaps) HasSplitedTVSeasons() bool {
	seenAniDBId := map[string]struct{}{}
	for i := range ms {
		m := &ms[i]
		if m.TVDBSeason < 1 || m.AniDBSeason < 1 {
			continue
		}
		if _, seen := seenAniDBId[m.AniDBId]; seen {
			return true
		}
		seenAniDBId[m.AniDBId] = struct{}{}
	}
	return false
}

func (ms AniDBTVDBEpisodeMaps) AreAbsoluteEpisode(episodes ...int) bool {
	hasAbsoluteOrder := false
	var largestNormalEpisode int
	for _, m := range ms {
		if m.HasAbsoluteOrder() {
			hasAbsoluteOrder = true
			continue
		} else if m.End != 0 && largestNormalEpisode < m.End {
			largestNormalEpisode = m.End
		} else if m.Start != 0 && largestNormalEpisode < m.Start {
			largestNormalEpisode = m.Start
		}
	}
	if !hasAbsoluteOrder {
		return false
	}
	for _, episode := range slices.Backward(episodes) {
		if episode > largestNormalEpisode {
			return true
		}
	}
	return false
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

func (ms AniDBTVDBEpisodeMaps) GetAbsoluteOrderSeasonMap() *AniDBTVDBEpisodeMap {
	for i := range ms {
		m := &ms[i]
		if m.HasAbsoluteOrder() {
			return m
		}
	}
	return nil
}

var query_get_tvdb_episode_maps_by_anidbid = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(TVDBEpisodeMapColumns...),
	TVDBEpisodeMapTableName,
	TVDBEpisodeMapColumn.AniDBId,
)
var query_get_tvdb_episode_maps_by_anidbid_with_related = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = (SELECT %s FROM %s WHERE %s = ? LIMIT 1)`,
	db.JoinColumnNames(TVDBEpisodeMapColumns...),
	TVDBEpisodeMapTableName,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapColumn.TVDBId,
	TVDBEpisodeMapTableName,
	TVDBEpisodeMapColumn.AniDBId,
)

type AniDBTVDBEpisodeMapsResult struct {
	AniDBTVDBEpisodeMaps
	groupedByAniDBId []tvdbEpisodeMapByAniDBId
	idxByAniDBId     map[string]int
	m                sync.Mutex
}

func (r *AniDBTVDBEpisodeMapsResult) Val() AniDBTVDBEpisodeMaps {
	return r.AniDBTVDBEpisodeMaps
}

func (r *AniDBTVDBEpisodeMapsResult) Len() int {
	return len(r.AniDBTVDBEpisodeMaps)
}

type tvdbEpisodeMapByAniDBId struct {
	AniDBId         string
	Part            int
	TVDBEpisodeMaps AniDBTVDBEpisodeMaps
}

func (r *AniDBTVDBEpisodeMapsResult) GroupByAniDBId() []tvdbEpisodeMapByAniDBId {
	r.m.Lock()
	defer r.m.Unlock()

	if r.groupedByAniDBId != nil {
		return r.groupedByAniDBId
	}
	groupedByAniDBId := []tvdbEpisodeMapByAniDBId{}
	idx := -1
	partByTvdbSeason := map[string]int{}
	for _, m := range r.AniDBTVDBEpisodeMaps {
		if idx == -1 || groupedByAniDBId[idx].AniDBId != m.AniDBId {
			idx++
		}
		if len(groupedByAniDBId) == idx {
			part := 0
			key := strconv.Itoa(m.TVDBSeason)
			if m.TVDBSeason > 0 && m.AniDBSeason > 0 {
				if lastPart, ok := partByTvdbSeason[key]; ok {
					partByTvdbSeason[key] = lastPart + 1
				} else {
					partByTvdbSeason[key] = 1
				}
				part = partByTvdbSeason[key]
			}

			groupedByAniDBId = append(groupedByAniDBId, tvdbEpisodeMapByAniDBId{
				AniDBId: m.AniDBId,
				Part:    part,
			})
		}
		groupedByAniDBId[idx].TVDBEpisodeMaps = append(groupedByAniDBId[idx].TVDBEpisodeMaps, m)
	}
	r.groupedByAniDBId = groupedByAniDBId
	r.idxByAniDBId = map[string]int{}
	for idx := range groupedByAniDBId {
		r.idxByAniDBId[groupedByAniDBId[idx].AniDBId] = idx
	}
	return groupedByAniDBId
}

func (r *AniDBTVDBEpisodeMapsResult) GetByAniDBId(anidbId string) *tvdbEpisodeMapByAniDBId {
	if r.idxByAniDBId == nil {
		r.GroupByAniDBId()
	}
	if idx, ok := r.idxByAniDBId[anidbId]; ok {
		return &r.groupedByAniDBId[idx]
	}
	return nil
}

// single tv season for the splitted anime seasons
func (r *AniDBTVDBEpisodeMapsResult) HasJoinedTVSeasons() bool {
	groupedByAniDBId := r.GroupByAniDBId()
	for i := range groupedByAniDBId {
		if groupedByAniDBId[i].Part > 1 {
			return true
		}
	}
	return false
}

func GetTVDBEpisodeMaps(anidbId string, includeRelated bool) (*AniDBTVDBEpisodeMapsResult, error) {
	query := query_get_tvdb_episode_maps_by_anidbid
	if includeRelated {
		query = query_get_tvdb_episode_maps_by_anidbid_with_related
	}
	rows, err := db.Query(query, anidbId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maps := AniDBTVDBEpisodeMaps{}
	for rows.Next() {
		m := AniDBTVDBEpisodeMap{
			Before: AniDBTVDBEpisodeMapBefore{},
			Map:    AniDBTVDBEpisodeMapMap{},
		}
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
	maps.Sort()
	return &AniDBTVDBEpisodeMapsResult{AniDBTVDBEpisodeMaps: maps}, nil
}
