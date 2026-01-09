package animelists

import (
	"embed"
	"encoding/xml"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

//go:embed anime-list.override.xml
var overridesFS embed.FS

type AnimeListItemMapping struct {
	XMLName xml.Name `xml:"mapping"`
	// 0 for special, 1 for regular
	AniDBSeason string `xml:"anidbseason,attr"`
	TVDBSeason  string `xml:"tvdbseason,attr"`
	TMDBSeason  string `xml:"tmdbseason,attr"`
	// anidb episode
	Start string `xml:"start,attr"`
	// anidb episode
	End string `xml:"end,attr"`
	// add to anidb episode to get tmdb episode
	Offset string `xml:"offset,attr"`
	Value  string `xml:",chardata"`
}

type AnimeListItem struct {
	XMLName xml.Name `xml:"anime"`
	AniDBId string   `xml:"anidbid,attr"`
	TVDBId  string   `xml:"tvdbid,attr"`
	// 'a' for absolute order
	DefaultTVDBSeason string                 `xml:"defaulttvdbseason,attr"`
	EpisodeOffset     string                 `xml:"episodeoffset,attr"`
	IMDBId            string                 `xml:"imdbid,attr"`
	TMDBId            string                 `xml:"tmdbid,attr"`
	TMDBOffset        string                 `xml:"tmdboffset,attr"`
	TMDBSeason        string                 `xml:"tmdbseason,attr"` // 'a' for absolute order
	TMDBTv            string                 `xml:"tmdbtv,attr"`
	Name              string                 `xml:"name"`
	Before            string                 `xml:"before"`
	Mappings          []AnimeListItemMapping `xml:"mapping-list>mapping"`
}

func (a *AnimeListItem) mergeOverride(o *AnimeListItem) {
	if o.TVDBId != "" && a.TVDBId != o.TVDBId {
		a.TVDBId = o.TVDBId
	}
	if o.DefaultTVDBSeason != "" && a.DefaultTVDBSeason != o.DefaultTVDBSeason {
		a.DefaultTVDBSeason = o.DefaultTVDBSeason
	}
	if o.EpisodeOffset != "" && a.EpisodeOffset != o.EpisodeOffset {
		a.EpisodeOffset = o.EpisodeOffset
	}
	if o.IMDBId != "" && a.IMDBId != o.IMDBId {
		a.IMDBId = o.IMDBId
	}
	if o.TMDBId != "" && a.TMDBId != o.TMDBId {
		a.TMDBId = o.TMDBId
	}
	if o.Name != "" && a.Name != o.Name {
		a.Name = o.Name
	}
	if o.Before != "" && a.Before != o.Before {
		a.Before = o.Before
	}
	if len(o.Mappings) > 0 {
		if len(a.Mappings) == 0 {
			a.Mappings = o.Mappings
		} else {
			for _, om := range o.Mappings {
				found := false
				for i, am := range a.Mappings {
					if am.AniDBSeason == om.AniDBSeason && am.TVDBSeason == om.TVDBSeason {
						orig := a.Mappings[i]
						if orig.Value != "" && om.Value == "" {
							om.Value = orig.Value
						}
						a.Mappings[i] = om
						found = true
						break
					}
				}
				if !found {
					a.Mappings = append(a.Mappings, om)
				}
			}
		}

		slices.SortFunc(a.Mappings, func(x, y AnimeListItemMapping) int {
			return util.SafeParseInt(x.TVDBSeason, -1) - util.SafeParseInt(y.TVDBSeason, -1)
		})
	}
}

func (a *AnimeListItem) Equal(b *AnimeListItem) bool {
	if a.AniDBId != b.AniDBId ||
		a.TVDBId != b.TVDBId ||
		a.DefaultTVDBSeason != b.DefaultTVDBSeason ||
		a.EpisodeOffset != b.EpisodeOffset ||
		a.IMDBId != b.IMDBId ||
		a.TMDBId != b.TMDBId ||
		a.Name != b.Name ||
		a.Before != b.Before {
		return false
	}
	if len(a.Mappings) != len(b.Mappings) {
		return false
	}
	for i := range a.Mappings {
		am, bm := &a.Mappings[i], &b.Mappings[i]
		if am.AniDBSeason != bm.AniDBSeason ||
			am.TVDBSeason != bm.TVDBSeason ||
			am.Start != bm.Start ||
			am.End != bm.End ||
			am.Offset != bm.Offset ||
			am.Value != bm.Value {
			return false
		}
	}
	return true
}

func PrepareAniDBTVDBEpisodeMaps(tvdbId string, items []AnimeListItem) *anidb.AniDBTVDBEpisodeMapsResult {
	tvdbMaps := []anidb.AniDBTVDBEpisodeMap{}

	// KEY: tvdb_season, VAL: anidb_part
	lastAniDBPartByTVDBSeason := map[string]int{}
	// KEY: tvdb_season:anidb_part, VAL: tvdb_map_idx
	seenPartMap := map[string]int{}

	for _, item := range items {
		episodeOffset := 0
		if item.EpisodeOffset != "" {
			if offset, err := strconv.Atoi(item.EpisodeOffset); err != nil {
				log.Error("failed to parse episodeoffset", "error", err, "item", item)
				continue
			} else {
				episodeOffset = offset
			}
		}

		seenMap := map[string]int{}
		absoluteKey := "1:-1"

		if item.DefaultTVDBSeason == "a" {
			tvdbMap := anidb.AniDBTVDBEpisodeMap{
				AniDBId:     item.AniDBId,
				TVDBId:      item.TVDBId,
				AniDBSeason: 1,
				TVDBSeason:  -1,
				Offset:      episodeOffset,
				Start:       1,
			}

			tvdbMaps = append(tvdbMaps, tvdbMap)
			seenMap[absoluteKey] = len(tvdbMaps) - 1
		} else {
			defaultTVDBSeason, err := strconv.Atoi(item.DefaultTVDBSeason)
			if err != nil {
				log.Error("failed to parse defaulttvdbseason", "error", err, "item", item)
				continue
			}

			tvdbMap := anidb.AniDBTVDBEpisodeMap{
				AniDBId:     item.AniDBId,
				TVDBId:      item.TVDBId,
				AniDBSeason: 1,
				TVDBSeason:  defaultTVDBSeason,
				Offset:      episodeOffset,
			}

			tvdbMaps = append(tvdbMaps, tvdbMap)
			seenMap["1:"+item.DefaultTVDBSeason] = len(tvdbMaps) - 1

			switch item.DefaultTVDBSeason {
			case "a", "0":
			default:
				anidbPart := 1
				if lastPart, ok := lastAniDBPartByTVDBSeason[item.DefaultTVDBSeason]; ok {
					anidbPart = lastPart + 1
				}
				lastAniDBPartByTVDBSeason[item.DefaultTVDBSeason] = anidbPart
				seenPartMap[item.DefaultTVDBSeason+":"+strconv.Itoa(anidbPart)] = len(tvdbMaps) - 1

				if anidbPart > 1 {
					tvdbMap := &tvdbMaps[len(tvdbMaps)-1]
					tvdbMap.Start = 1

					lastPart := anidbPart - 1
					lastPartKey := item.DefaultTVDBSeason + ":" + strconv.Itoa(lastPart)
					if lastPartIdx, ok := seenPartMap[lastPartKey]; ok {
						lastPartTvdbMap := &tvdbMaps[lastPartIdx]
						if lastPartTvdbMap.Start == 0 {
							lastPartTvdbMap.Start = 1
						}
						if lastPartTvdbMap.End == 0 {
							lastPartTvdbMap.End = tvdbMap.Offset - lastPartTvdbMap.Offset
						}
					}
				}
			}
		}

		if item.Before != "" {
			key := "1:" + item.DefaultTVDBSeason
			if item.DefaultTVDBSeason == "a" {
				key = absoluteKey
			}

			tvdbMap := &tvdbMaps[seenMap[key]]
			tvdbMap.Before = anidb.AniDBTVDBEpisodeMapBefore{}
			err := tvdbMap.Before.Scan(item.Before)
			if err != nil {
				log.Error("failed to parse before", "error", err, "item", item)
				continue
			}
		}

		for i := range item.Mappings {
			m := &item.Mappings[i]

			key := m.AniDBSeason + ":" + m.TVDBSeason

			var tvdbMap *anidb.AniDBTVDBEpisodeMap
			tvdbMapIdx, seen := seenMap[key]
			if seen {
				tvdbMap = &tvdbMaps[tvdbMapIdx]
			} else {
				tvdbMap = &anidb.AniDBTVDBEpisodeMap{
					AniDBId: item.AniDBId,
					TVDBId:  item.TVDBId,
				}
			}

			switch m.AniDBSeason {
			case "0":
				tvdbMap.AniDBSeason = 0
			case "1":
				tvdbMap.AniDBSeason = 1
			}

			if m.TVDBSeason == "" && m.TMDBSeason != "" {
				// TODO: process tmdb data
				continue
			}

			tvdbSeason, err := strconv.Atoi(m.TVDBSeason)
			if err != nil {
				log.Error("failed to parse tvdbseason", "error", err, "item", item, "mapping", m)
				continue
			}
			tvdbMap.TVDBSeason = tvdbSeason

			if m.Start != "" {
				start, err := strconv.Atoi(m.Start)
				if err != nil {
					log.Error("failed to parse start", "error", err, "item", item, "mapping", m)
					continue
				}
				tvdbMap.Start = start
			}

			if m.End != "" {
				end, err := strconv.Atoi(m.End)
				if err != nil {
					log.Error("failed to parse end", "error", err, "item", item, "mapping", m)
					continue
				}
				tvdbMap.End = end
			}

			if m.Offset != "" {
				offset, err := strconv.Atoi(m.Offset)
				if err != nil {
					log.Error("failed to parse offset", "error", err, "item", item, "mapping", m)
					continue
				}
				tvdbMap.Offset = offset
			}

			if m.Value != "" {
				tvdbMap.Map = anidb.AniDBTVDBEpisodeMapMap{}
				err := tvdbMap.Map.Scan(m.Value)
				if err != nil {
					log.Error("failed to parse value", "error", err, "item", item, "mapping", m)
					continue
				}
			}

			if !seen {
				if tvdbMap.TVDBSeason == 2 && m.AniDBSeason == "1" {
					seasonOneKey := m.AniDBSeason + ":1"
					if idx, seenSeasonOne := seenMap[seasonOneKey]; seenSeasonOne {
						seasonOneMap := &tvdbMaps[idx]
						if seasonOneMap.Start == 0 && seasonOneMap.End == 0 {
							if seasonOneMap.Offset == 0 {
								seasonOneMap.Offset = tvdbMap.Start - 1 + tvdbMap.Offset
							}
							seasonOneMap.Start = 1
							seasonOneMap.End = tvdbMap.Start - 1
						}
					} else {
						seasonOneMap := anidb.AniDBTVDBEpisodeMap{
							AniDBId:     item.AniDBId,
							TVDBId:      item.TVDBId,
							AniDBSeason: tvdbMap.AniDBSeason,
							TVDBSeason:  1,
							Offset:      tvdbMap.Start - 1 + tvdbMap.Offset,
							Start:       1,
							End:         tvdbMap.Start - 1,
						}
						tvdbMaps = append(tvdbMaps, seasonOneMap)
						seenMap[seasonOneKey] = len(tvdbMaps) - 1
					}
				}
				tvdbMaps = append(tvdbMaps, *tvdbMap)
				seenMap[key] = len(tvdbMaps) - 1
			}

			if absMapIdx, ok := seenMap[absoluteKey]; ok {
				absMap := &tvdbMaps[absMapIdx]
				if tvdbMap.End == 0 || absMap.End < tvdbMap.End {
					absMap.End = tvdbMap.End
				}
			}
		}
	}

	return &anidb.AniDBTVDBEpisodeMapsResult{AniDBTVDBEpisodeMaps: tvdbMaps}
}

func SyncDataset() error {
	log := logger.Scoped("animelists/dataset")

	getItemKey := func(item *AnimeListItem) string {
		return item.AniDBId
	}

	overrideByAniDBId := map[string]AnimeListItem{}
	overrideFile, err := overridesFS.Open("anime-list.override.xml")
	if err != nil {
		return err
	}
	overrideReader := util.NewXMLDatasetReader(&util.XMLDatasetReaderConfig[AnimeListItem, AnimeListItem]{
		File:        overrideFile,
		GetItemKey:  getItemKey,
		ItemTagName: "anime",
		ListTagName: "anime-list",
		Log:         log,
	})

	for {
		item := overrideReader.NextItem()
		if item == nil {
			break
		}
		overrideByAniDBId[item.AniDBId] = *item
	}

	regexDigits := regexp.MustCompile(`^\d+$`)
	byTVDBId := map[string][]AnimeListItem{}

	ds := util.NewXMLDataset(&util.XMLDatasetConfig[AnimeListItem, AnimeListItem]{
		DatasetConfig: util.DatasetConfig{
			DownloadDir: path.Join(config.DataDir, "anime-lists"),
			URL:         "https://github.com/Anime-Lists/anime-lists/raw/refs/heads/master/anime-list.xml",
			Log:         log,
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-24 * time.Hour))
			},
		},
		ListTagName: "anime-list",
		ItemTagName: "anime",
		NoDiff:      true,
		GetItemKey:  getItemKey,
		IsItemEqual: func(a, b *AnimeListItem) bool {
			return a.Equal(b)
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[AnimeListItem]{
			BatchSize: 500,
			Log:       log,
			Upsert: func(items []AnimeListItem) error {
				for i := range items {
					item := &items[i]
					// TODO: upsert into `anime_id_map` for IMDB and TMDB
					if !regexDigits.MatchString(item.TVDBId) {
						switch strings.ToLower(item.TVDBId) {
						case "movie":
							err := anidb.SetTitleType(item.AniDBId, anime.AnimeIdMapTypeMovie)
							if err != nil {
								log.Error("failed to set title type for movie", "error", err, "item", item)
							}
						}
						continue
					}
					if byTVDBId[item.TVDBId] == nil {
						byTVDBId[item.TVDBId] = []AnimeListItem{}
					}
					if override, ok := overrideByAniDBId[item.AniDBId]; ok {
						item.mergeOverride(&override)
					}
					byTVDBId[item.TVDBId] = append(byTVDBId[item.TVDBId], *item)
				}
				return nil
			},
			SleepDuration: 200 * time.Millisecond,
		}),
	})

	if err := ds.Process(); err != nil {
		return err
	}

	for tvdbId, items := range byTVDBId {
		tvdbMaps := PrepareAniDBTVDBEpisodeMaps(tvdbId, items)
		if err := anidb.UpsertTVDBEpisodeMaps(tvdbMaps.Val()); err != nil {
			return err
		}
	}

	log.Info("rebuilding anidb_title fts...")
	if err := anidb.RebuildTitleFTS(); err != nil {
		return err
	}

	return nil
}
