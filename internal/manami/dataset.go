package manami

import (
	"encoding/json"
	"iter"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type AnimeItemType string

const (
	AnimeItemTypeTV      AnimeItemType = "TV"
	AnimeItemTypeMovie   AnimeItemType = "MOVIE"
	AnimeItemTypeOVA     AnimeItemType = "OVA"
	AnimeItemTypeONA     AnimeItemType = "ONA"
	AnimeItemTypeSpecial AnimeItemType = "SPECIAL"
	AnimeItemTypeUnknown AnimeItemType = "UNKNOWN"
)

type AnimeItem struct {
	Sources     []string      `json:"sources"`
	Title       string        `json:"title"`
	Type        AnimeItemType `json:"type"`
	Episodes    int           `json:"episodes"`
	AnimeSeason struct {
		Season string `json:"season"`
		Year   int    `json:"year"`
	} `json:"animeSeason"`
}

type ParsedAnime struct {
	Id   string
	Type AnimeItemType
	Year int
	Ids  struct {
		AniList     string `json:"anilist"`
		AniDB       string `json:"anidb"`
		AniSearch   string `json:"anisearch"`
		AnimePlanet string `json:"animeplanet"`
		Kitsu       string `json:"kitsu"`
		LiveChart   string `json:"livechart"`
		MAL         string `json:"mal"`
		NotifyMoe   string `json:"notifymoe"`
	}
}

func (a *ParsedAnime) Equal(b *ParsedAnime) bool {
	if a.Id != b.Id || a.Type != b.Type || a.Year != b.Year {
		return false
	}

	if a.Ids.AniList != b.Ids.AniList ||
		a.Ids.AniDB != b.Ids.AniDB ||
		a.Ids.AniSearch != b.Ids.AniSearch ||
		a.Ids.AnimePlanet != b.Ids.AnimePlanet ||
		a.Ids.Kitsu != b.Ids.Kitsu ||
		a.Ids.LiveChart != b.Ids.LiveChart ||
		a.Ids.MAL != b.Ids.MAL ||
		a.Ids.NotifyMoe != b.Ids.NotifyMoe {
		return false
	}

	return true

}

type AnimeDatabase struct {
	LastUpdate string      `json:"lastUpdate"` // 2006-01-02
	Data       []AnimeItem `json:"data"`
}

func SyncDataset() error {
	log := logger.Scoped("manami/dataset")

	ds := util.NewJSONDataset(&util.JSONDatasetConfig[ParsedAnime]{
		DatasetConfig: util.DatasetConfig{
			DownloadDir: path.Join(config.DataDir, "manami-anime-database"),
			URL:         "https://github.com/manami-project/anime-offline-database/releases/download/latest/anime-offline-database-minified.json",
			DownloadHeaders: map[string]string{
				"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			},
			Log: log,
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-6 * 24 * time.Hour))
			},
		},
		GetSeq: func(blob []byte) (iter.Seq[*ParsedAnime], error) {
			data := AnimeDatabase{}
			err := json.Unmarshal(blob, &data)
			if err != nil {
				return nil, err
			}
			return func(yield func(*ParsedAnime) bool) {
				for i := range data.Data {
					item := &data.Data[i]

					parsed := ParsedAnime{
						Type: item.Type,
						Year: item.AnimeSeason.Year,
					}

					for i := range item.Sources {
						source, err := url.Parse(item.Sources[i])
						if err != nil {
							continue
						}
						switch source.Hostname() {
						case "anidb.net":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.AniDB = id
						case "anilist.co":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.AniList = id
						case "anime-planet.com":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.AnimePlanet = id
						case "anisearch.com":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.AniSearch = id
						case "kitsu.app":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.Kitsu = id
						case "livechart.me":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.LiveChart = id
						case "myanimelist.net":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.MAL = id
						case "notify.moe":
							id := strings.TrimSuffix(strings.TrimPrefix(source.Path, "/anime/"), "/")
							parsed.Ids.NotifyMoe = id
						}
					}

					if parsed.Ids.AniList != "" {
						parsed.Id = anime.IdMapColumn.AniList + ":" + parsed.Ids.AniList
					} else if parsed.Ids.MAL != "" {
						parsed.Id = anime.IdMapColumn.MAL + ":" + parsed.Ids.MAL
					} else if parsed.Ids.Kitsu != "" {
						parsed.Id = anime.IdMapColumn.Kitsu + ":" + parsed.Ids.Kitsu
					} else if parsed.Ids.AniDB != "" {
						parsed.Id = anime.IdMapColumn.AniDB + ":" + parsed.Ids.AniDB
					} else if parsed.Ids.AniSearch != "" {
						parsed.Id = anime.IdMapColumn.AniSearch + ":" + parsed.Ids.AniSearch
					} else if parsed.Ids.AnimePlanet != "" {
						parsed.Id = anime.IdMapColumn.AnimePlanet + ":" + parsed.Ids.AnimePlanet
					} else if parsed.Ids.LiveChart != "" {
						parsed.Id = anime.IdMapColumn.LiveChart + ":" + parsed.Ids.LiveChart
					} else if parsed.Ids.NotifyMoe != "" {
						parsed.Id = anime.IdMapColumn.NotifyMoe + ":" + parsed.Ids.NotifyMoe
					}

					if parsed.Id == "" {
						continue
					}

					if !yield(&parsed) {
						return
					}
				}
			}, nil
		},
		GetItemKey: func(item *ParsedAnime) string {
			return item.Id
		},
		IsItemEqual: func(a, b *ParsedAnime) bool {
			return a.Equal(b)
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[ParsedAnime]{
			BatchSize: 200,
			Log:       log,
			Upsert: func(items []ParsedAnime) error {
				idMapsByAnchor := map[string][]anime.AnimeIdMap{}
				yearByAniDBId := map[string]int{}
				for i := range items {
					item := &items[i]
					anchorColumn, _, _ := strings.Cut(item.Id, ":")
					idMap := anime.AnimeIdMap{
						Type:        anime.AnimeIdMapTypeUnknown,
						AniList:     item.Ids.AniList,
						AniDB:       item.Ids.AniDB,
						AniSearch:   item.Ids.AniSearch,
						AnimePlanet: item.Ids.AnimePlanet,
						Kitsu:       item.Ids.Kitsu,
						LiveChart:   item.Ids.LiveChart,
						MAL:         item.Ids.MAL,
						NotifyMoe:   item.Ids.NotifyMoe,
					}
					switch item.Type {
					case AnimeItemTypeTV:
						idMap.Type = anime.AnimeIdMapTypeTV
					case AnimeItemTypeMovie:
						idMap.Type = anime.AnimeIdMapTypeMovie
					case AnimeItemTypeOVA:
						idMap.Type = anime.AnimeIdMapTypeOVA
					case AnimeItemTypeONA:
						idMap.Type = anime.AnimeIdMapTypeONA
					case AnimeItemTypeSpecial:
						idMap.Type = anime.AnimeIdMapTypeSpecial
					}
					idMapsByAnchor[anchorColumn] = append(idMapsByAnchor[anchorColumn], idMap)
					if idMap.AniDB != "" && item.Year != 0 {
						yearByAniDBId[idMap.AniDB] = item.Year
					}
				}
				for anchorColumn, idMaps := range idMapsByAnchor {
					if anchorColumn == "" {
						log.Warn("skipping id maps with no anchor column", "count", len(idMaps))
						continue
					}
					anime.BulkRecordIdMaps(idMaps, anchorColumn)
					log.Info("recorded id maps", "anchor_column", anchorColumn, "count", len(idMaps))
				}
				if len(yearByAniDBId) > 0 {
					for anidbId, year := range yearByAniDBId {
						err := anidb.SetTitleYear(anidbId, year)
						if err != nil {
							log.Error("failed to set title year", "anidb_id", anidbId, "year", year, "error", err)
							return err
						}
					}
					log.Info("set anidb title years", "count", len(yearByAniDBId))
				}
				return nil
			},
			SleepDuration: 200 * time.Millisecond,
		}),
	})

	if err := ds.Process(); err != nil {
		return err
	}

	log.Info("rebuilding anidb_title fts...")
	if err := anidb.RebuildTitleFTS(); err != nil {
		return err
	}

	return nil
}
