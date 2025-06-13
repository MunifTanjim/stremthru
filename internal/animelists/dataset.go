package animelists

import (
	"encoding/xml"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type AnimeListItemMapping struct {
	XMLName xml.Name `xml:"mapping"`
	// 0 for special, 1 for regular
	AniDBSeason string `xml:"anidbseason,attr"`
	TVDBSeason  string `xml:"tvdbseason,attr"`
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
	Name              string                 `xml:"name"`
	Before            string                 `xml:"before"`
	Mappings          []AnimeListItemMapping `xml:"mapping-list>mapping"`
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

func parseIntId(input string) int {
	if input == "" {
		return 0
	}
	output, err := strconv.Atoi(input)
	if err != nil {
		return 0
	}
	return output
}

func parseStringId(input string) string {
	switch input {
	case "unknown", "movie", "hentai", "web":
		return ""
	default:
		return input
	}
}

func processAnimeListItemsForTVDBId(tvdbId string, items []AnimeListItem) []anidb.AniDBTVDBEpisodeMap {
	tvdbMaps := []anidb.AniDBTVDBEpisodeMap{}

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

		if item.DefaultTVDBSeason == "a" {
			tvdbMap := anidb.AniDBTVDBEpisodeMap{
				AniDBId:     item.AniDBId,
				TVDBId:      item.TVDBId,
				AniDBSeason: 1,
				TVDBSeason:  -1,
				Offset:      episodeOffset,
			}

			tvdbMaps = append(tvdbMaps, tvdbMap)
			seenMap["1:a"] = len(tvdbMaps) - 1

			// hasSeasonOneMapping := slices.IndexFunc(item.Mappings, func(m AnimeListItemMapping) bool {
			// 	return m.TVDBSeason == "1"
			// }) != -1
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

			if m.AniDBSeason == "0" {
				tvdbMap.AniDBSeason = 0
			} else if m.AniDBSeason == "1" {
				tvdbMap.AniDBSeason = 1
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
				tvdbMaps = append(tvdbMaps, *tvdbMap)
				seenMap[key] = len(tvdbMaps) - 1
			}
		}
	}

	return tvdbMaps
}

func SyncDataset() error {
	log := logger.Scoped("animelists/dataset")

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
		GetItemKey: func(item *AnimeListItem) string {
			return item.AniDBId
		},
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
						continue
					}
					if byTVDBId[item.TVDBId] == nil {
						byTVDBId[item.TVDBId] = []AnimeListItem{}
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
		tvdbMaps := processAnimeListItemsForTVDBId(tvdbId, items)
		if err := anidb.UpsertTVDBEpisodeMaps(tvdbMaps); err != nil {
			return err
		}
	}

	return nil
}
