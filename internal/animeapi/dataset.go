package animeapi

import (
	"maps"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/kitsu"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type Mapping struct {
	Title            string `json:"title"`
	AniDB            int    `json:"anidb"`
	AniList          int    `json:"anilist"`
	AnimeNewsNetwork string `json:"animenewsnetwork"`
	AnimePlanet      string `json:"animeplanet"`
	AniSearch        int    `json:"anisearch"`
	Annict           int    `json:"annict"`
	IMDB             string `json:"imdb"`
	Kaize            string `json:"kaize"`
	KaizeId          int    `json:"kaize_id"`
	Kitsu            int    `json:"kitsu"`
	LiveChart        int    `json:"livechart"`
	MyAnimeList      int    `json:"myanimelist"`
	Nautiljon        string `json:"nautiljon"`
	NautiljonId      int    `json:"nautiljon_id"`
	Notify           string `json:"notify"`
	Otakotaku        int    `json:"otakotaku"`
	Shikimori        int    `json:"shikimori"`
	Shoboi           int    `json:"shoboi"`
	Silveryasha      int    `json:"silveryasha"`
	Simkl            int    `json:"simkl"`
	TMDB             int    `json:"themoviedb"`
	Trakt            int    `json:"trakt"`
	TraktType        string `json:"trakt_type"` // movies / shows
	TraktSeason      int    `json:"trakt_season"`
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
	if input == "unknown" {
		return ""
	}
	return input
}

func SyncDataset() error {
	lastUpdatedAt, err := getLastUpdated()
	if err != nil {
		return err
	}

	Kitsu := kitsu.GetSystemKitsu()

	ds := util.NewTSVDataset(&util.TSVDatasetConfig[Mapping]{
		DatasetConfig: util.DatasetConfig{
			DownloadDir: path.Join(config.DataDir, "animeapi"),
			URL:         "https://raw.githubusercontent.com/nattadasu/animeApi/refs/heads/v3/database/animeapi.tsv",
			Log:         datasetLog,
			GetDownloadFileTime: func() time.Time {
				return lastUpdatedAt
			},
			IsStale: func(t time.Time) bool {
				return t.Before(lastUpdatedAt)
			},
		},
		HasHeaders: true,
		IsValidHeaders: func(headers []string) bool {
			return slices.Equal(headers, []string{
				"title",
				"anidb",
				"anilist",
				"animenewsnetwork",
				"animeplanet",
				"anisearch",
				"annict",
				"imdb",
				"kaize",
				"kaize_id",
				"kitsu",
				"livechart",
				"myanimelist",
				"nautiljon",
				"nautiljon_id",
				"notify",
				"otakotaku",
				"shikimori",
				"shoboi",
				"silveryasha",
				"simkl",
				"themoviedb",
				"trakt",
				"trakt_type",
				"trakt_season",
			})
		},
		GetRowKey: func(row []string) string {
			return row[0]
		},
		ParseRow: func(row []string) (*Mapping, error) {
			m := Mapping{
				Title:            row[0],
				AniDB:            parseIntId(row[1]),
				AniList:          parseIntId(row[2]),
				AnimeNewsNetwork: parseStringId(row[3]),
				AnimePlanet:      parseStringId(row[4]),
				AniSearch:        parseIntId(row[5]),
				Annict:           parseIntId(row[6]),
				IMDB:             parseStringId(row[7]),
				Kaize:            parseStringId(row[8]),
				KaizeId:          parseIntId(row[9]),
				Kitsu:            parseIntId(row[10]),
				LiveChart:        parseIntId(row[11]),
				MyAnimeList:      parseIntId(row[12]),
				Nautiljon:        parseStringId(row[13]),
				NautiljonId:      parseIntId(row[14]),
				Notify:           parseStringId(row[15]),
				Otakotaku:        parseIntId(row[16]),
				Shikimori:        parseIntId(row[17]),
				Shoboi:           parseIntId(row[18]),
				Silveryasha:      parseIntId(row[19]),
				Simkl:            parseIntId(row[20]),
				TMDB:             parseIntId(row[21]),
				Trakt:            parseIntId(row[22]),
				TraktType:        parseStringId(row[23]),
				TraktSeason:      parseIntId(row[24]),
			}
			return &m, nil
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[Mapping]{
			BatchSize: 500,
			Log:       datasetLog,
			Upsert: func(items []Mapping) error {
				typeByAnilistId := map[int]anime.AnimeIdMapType{}
				typeByIMDBId := map[string]anime.AnimeIdMapType{}
				typeByTraktId := map[int]anime.AnimeIdMapType{}
				typeByKitsuId := map[int]anime.AnimeIdMapType{}
				anilistIdsMissingType := []int{}
				kitsuIdsMissingTypes := []int{}
				imdbIdsMissingType := []string{}
				for i := range items {
					item := &items[i]
					if item.TraktType == "movies" {
						typeByTraktId[item.Trakt] = anime.AnimeIdMapTypeMovie
					} else if item.TraktType == "shows" {
						typeByTraktId[item.Trakt] = anime.AnimeIdMapTypeTV
					} else if item.AniList != 0 {
						anilistIdsMissingType = append(anilistIdsMissingType, item.AniList)
					} else if Kitsu != nil && item.Kitsu != 0 {
						kitsuIdsMissingTypes = append(kitsuIdsMissingTypes, item.Kitsu)
					} else if item.IMDB != "" {
						imdbIdsMissingType = append(imdbIdsMissingType, item.IMDB)
					}
				}

				if byAnilistId, err := anime.GetTypeByAnilistIds(anilistIdsMissingType); err == nil {
					if count := len(byAnilistId); count > 0 {
						datasetLog.Debug("found anime type by anilist ids", "count", count)
						maps.Copy(typeByAnilistId, byAnilistId)
						anilistIdsMissingType = slices.DeleteFunc(anilistIdsMissingType, func(id int) bool {
							t, ok := byAnilistId[id]
							return ok && t != ""
						})
					}
				}
				if infos, err := anilist.FetchAnimeMediaFormatInfo(anilistIdsMissingType); err == nil {
					if count := len(infos); count > 0 {
						datasetLog.Debug("fetched anime media format from anilist", "count", count)
						for i := range infos {
							info := &infos[i]
							typeByAnilistId[info.Id] = anime.AnimeIdMapType(info.Format)
						}
					}
				} else {
					datasetLog.Error("failed to fetch anime media format from anilist", "error", err, "count", len(anilistIdsMissingType))
				}

				if Kitsu != nil {
					if byKitsuId, err := anime.GetTypeByKitsuIds(kitsuIdsMissingTypes); err == nil {
						if count := len(byKitsuId); count > 0 {
							datasetLog.Debug("found anime type by kitsu ids", "count", count)
							maps.Copy(typeByKitsuId, byKitsuId)
							kitsuIdsMissingTypes = slices.DeleteFunc(kitsuIdsMissingTypes, func(id int) bool {
								t, ok := byKitsuId[id]
								return ok && t != ""
							})
						}
					}
					if byKitsuId, err := Kitsu.GetAnimeTypeByIds(&kitsu.GetAnimeTypeByIdsParams{
						Ids: kitsuIdsMissingTypes,
					}); err == nil {
						if count := len(byKitsuId.Data); count > 0 {
							datasetLog.Debug("fetched anime type from kitsu", "count", count)
							for id, animeType := range byKitsuId.Data {
								switch animeType {
								case kitsu.AnimeSubtypeMovie:
									typeByKitsuId[id] = anime.AnimeIdMapTypeMovie
								case kitsu.AnimeSubtypeMusic:
									typeByKitsuId[id] = anime.AnimeIdMapTypeMusic
								case kitsu.AnimeSubtypeONA:
									typeByKitsuId[id] = anime.AnimeIdMapTypeONA
								case kitsu.AnimeSubtypeOVA:
									typeByKitsuId[id] = anime.AnimeIdMapTypeOVA
								case kitsu.AnimeSubtypeSpecial:
									typeByKitsuId[id] = anime.AnimeIdMapTypeSpecial
								case kitsu.AnimeSubtypeTV:
									typeByKitsuId[id] = anime.AnimeIdMapTypeTV
								}
							}
						}
					} else {
						datasetLog.Error("failed to fetch anime type from kitsu", "error", err, "count", len(kitsuIdsMissingTypes))
					}
				}

				if typeMap, err := imdb_title.GetTypeByIds(imdbIdsMissingType); err == nil {
					if count := len(typeMap); count > 0 {
						datasetLog.Debug("found type for IMDB titles", "count", len(typeMap))
						for imdbId, t := range typeMap {
							switch t {
							case imdb_title.IMDBTitleTypeMovie,
								imdb_title.IMDBTitleTypeTvMovie:

								typeByIMDBId[imdbId] = anime.AnimeIdMapTypeMovie

							case imdb_title.IMDBTitleTypeShort,
								imdb_title.IMDBTitleTypeTvShort,
								imdb_title.IMDBTitleTypeTvSeries,
								imdb_title.IMDBTitleTypeTvMiniSeries,
								imdb_title.IMDBTitleTypeTvSpecial:

								typeByIMDBId[imdbId] = anime.AnimeIdMapTypeTV
							}
						}
					}
				} else {
					datasetLog.Error("failed to fetch IMDB title types", "error", err, "count", len(imdbIdsMissingType))
				}

				idMapsByAnchor := map[string][]anime.AnimeIdMap{}
				for i := range items {
					item := &items[i]
					anchorColumn := ""
					if item.AniList != 0 {
						anchorColumn = anime.IdMapColumn.AniList
					} else if item.MyAnimeList != 0 {
						anchorColumn = anime.IdMapColumn.MAL
					} else if item.Kitsu != 0 {
						anchorColumn = anime.IdMapColumn.Kitsu
					} else if item.AniDB != 0 {
						anchorColumn = anime.IdMapColumn.AniDB
					} else if item.AniSearch != 0 {
						anchorColumn = anime.IdMapColumn.AniSearch
					} else if item.AnimePlanet != "" {
						anchorColumn = anime.IdMapColumn.AnimePlanet
					} else if item.LiveChart != 0 {
						anchorColumn = anime.IdMapColumn.LiveChart
					} else if item.Notify != "" {
						anchorColumn = anime.IdMapColumn.NotifyMoe
					}
					idMap := anime.AnimeIdMap{
						Type:        "",
						AniList:     strconv.Itoa(item.AniList),
						AniDB:       strconv.Itoa(item.AniDB),
						AniSearch:   strconv.Itoa(item.AniSearch),
						AnimePlanet: item.AnimePlanet,
						IMDB:        item.IMDB,
						Kitsu:       strconv.Itoa(item.Kitsu),
						LiveChart:   strconv.Itoa(item.LiveChart),
						MAL:         strconv.Itoa(item.MyAnimeList),
						NotifyMoe:   item.Notify,
						TMDB:        strconv.Itoa(item.TMDB),
						TVDB:        "",
					}
					if t, found := typeByTraktId[item.Trakt]; found {
						idMap.Type = t
					} else if t, found := typeByAnilistId[item.AniList]; found {
						idMap.Type = t
					} else if t, found := typeByKitsuId[item.Kitsu]; found {
						idMap.Type = t
					} else if t, found := typeByIMDBId[item.IMDB]; found {
						idMap.Type = t
					}
					idMapsByAnchor[anchorColumn] = append(idMapsByAnchor[anchorColumn], idMap)
				}

				for anchorColumn, idMaps := range idMapsByAnchor {
					if anchorColumn == "" {
						datasetLog.Warn("skipping id maps with no anchor column", "count", len(idMaps))
						continue
					}
					anime.BulkRecordIdMaps(idMaps, anchorColumn)
					datasetLog.Info("synced id maps", "anchor_column", anchorColumn, "count", len(idMaps))
				}
				return nil
			},
			SleepDuration: 200 * time.Millisecond,
		}),
	})

	if err := ds.Process(); err != nil {
		return err
	}

	return nil
}
