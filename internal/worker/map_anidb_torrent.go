package worker

import (
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/agnivade/levenshtein"
	"github.com/madflojo/tasks"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
)

type torrentMap struct {
	anidbId      string
	seasonType   anidb.TorrentSeasonType
	season       int
	episodeStart int
	episodeEnd   int
	episodes     []int
}

func prepareAniDBTorrentMaps(tvdbMaps anidb.AniDBTVDBEpisodeMaps, titles anidb.AniDBTitles, tInfo torrent_info.TorrentInfo) ([]torrentMap, error) {
	tSeasonCount, tEpisodeCount := len(tInfo.Seasons), len(tInfo.Episodes)

	tMaps := []torrentMap{}

	tYearThreshold := 0
	if tInfo.Year != 0 {
		tYearThreshold = 1
		if tInfo.YearEnd != 0 {
			tYearThreshold = tInfo.YearEnd - tInfo.Year
		}
		tYearThreshold += 2
	}

	if tSeasonCount == 0 && tEpisodeCount == 0 {
		tYear := ""
		if tInfo.Year != 0 {
			tYear = strconv.Itoa(tInfo.Year)
		}

		slices.SortStableFunc(titles, func(a, b anidb.AniDBTitle) int {
			if tYear != "" {
				if a.Year == b.Year {
					return levenshtein.ComputeDistance(tInfo.Title, a.Value) - levenshtein.ComputeDistance(tInfo.Title, b.Value)
				}
				if a.Year == tYear {
					return -1
				}
				if b.Year == tYear {
					return 1
				}
			}
			return levenshtein.ComputeDistance(tInfo.Title, a.Value) - levenshtein.ComputeDistance(tInfo.Title, b.Value)
		})

		if len(titles) == 0 {
			return tMaps, nil
		}

		title := titles[0]

		if tInfo.Category == torrent_info.TorrentInfoCategoryMovie && tYear != "" && title.Year != "" && tYear != title.Year {
			return tMaps, nil
		}

		for _, anidbGroup := range tvdbMaps.GroupByAniDBId() {
			if anidbGroup.AniDBId != title.TId {
				continue
			}

			animeYear := titles.GetYear(anidbGroup.AniDBId)
			if animeYear != 0 && tInfo.Year != 0 {
				diff := animeYear - tInfo.Year
				if max(diff, -diff) > tYearThreshold {
					continue
				}
			}

			for i := range anidbGroup.TVDBEpisodeMaps {
				tvdbMap := &anidbGroup.TVDBEpisodeMaps[i]

				if title.Year != "" && tYear == title.Year {
					if tvdbMap.HasAbsoluteOrder() {
						tMap := torrentMap{
							anidbId:    title.TId,
							seasonType: anidb.TorrentSeasonTypeAbsolute,
							season:     tvdbMap.TVDBSeason,
						}
						tMap.episodeStart, tMap.episodeEnd = tvdbMap.Start+tvdbMap.Offset, tvdbMap.End+tvdbMap.Offset
						tMaps = append(tMaps, tMap)

						if season := util.SafeParseInt(title.Season, -1); season != -1 {
							tMap = torrentMap{
								anidbId:    title.TId,
								seasonType: anidb.TorrentSeasonTypeTypeAnime,
								season:     season,
							}
							tMap.episodeStart, tMap.episodeEnd = tvdbMap.Start, tvdbMap.End
							tMaps = append(tMaps, tMap)
						}
					} else if tvdbMap.IsAniDBRegularSeason() {
						tvTMap := torrentMap{
							anidbId:    title.TId,
							seasonType: anidb.TorrentSeasonTypeTV,
							season:     tvdbMap.TVDBSeason,
						}
						if tvdbMap.Start != 0 && tvdbMap.End != 0 {
							tvTMap.episodeStart, tvTMap.episodeEnd = tvdbMap.Start+tvdbMap.Offset, tvdbMap.End+tvdbMap.Offset
						}
						tMaps = append(tMaps, tvTMap)
					}
				} else if title.Year == "" {
					if tvdbMap.IsAniDBRegularSeason() {
						aniTMap := torrentMap{
							anidbId:    title.TId,
							seasonType: anidb.TorrentSeasonTypeTypeAnime,
							season:     util.SafeParseInt(title.Season, -1),
						}
						tvTMap := torrentMap{
							anidbId:    title.TId,
							seasonType: anidb.TorrentSeasonTypeTV,
							season:     tvdbMap.TVDBSeason,
						}
						if tvdbMap.Start != 0 && tvdbMap.End != 0 {
							tvTMap.episodeStart, tvTMap.episodeEnd = tvdbMap.Start+tvdbMap.Offset, tvdbMap.End+tvdbMap.Offset
						} else if len(tvdbMap.Map) > 0 {
							seenTVEpis := map[int]struct{}{}
							for aniEp, tvEps := range tvdbMap.Map {
								aniTMap.episodes = append(aniTMap.episodes, aniEp)
								for _, tvEp := range tvEps {
									if _, seen := seenTVEpis[tvEp]; !seen {
										tvTMap.episodes = append(tvTMap.episodes, tvEp)
									}
								}
							}
						}
						tMaps = append(tMaps, aniTMap, tvTMap)
					}
				}
			}
		}
		return tMaps, nil
	}

	if tSeasonCount != 0 {
		idSeen := map[string]struct{}{}

		hasEpisodes := tEpisodeCount > 0
		tFirstEpisode, tLastEpisode := -1, -1
		hasAbsoluteOrder := tvdbMaps.HasAbsoluteOrder()
		tEpisodesAreAbsolute := false
		if hasEpisodes {
			tFirstEpisode = tInfo.Episodes[0]
			tLastEpisode = tInfo.Episodes[len(tInfo.Episodes)-1]
			if hasAbsoluteOrder {
				tEpisodesAreAbsolute = tvdbMaps.AreAbsoluteEpisode(tInfo.Episodes...)
			}
		}

		minAnimeSeason, maxAnimeSeason := titles.SeasonBoundary()
		canBeAnimeSeason := tEpisodeCount != 1
		if canBeAnimeSeason {
			if !hasEpisodes && hasAbsoluteOrder {
				canBeAnimeSeason = false
			} else {
				for _, season := range tInfo.Seasons {
					if season < minAnimeSeason || maxAnimeSeason < season {
						canBeAnimeSeason = false
						break
					}
				}
			}
		}
		for _, anidbGroup := range tvdbMaps.GroupByAniDBId() {
			animeSeason := titles.GetSeason(anidbGroup.AniDBId)

			animeYear := titles.GetYear(anidbGroup.AniDBId)
			if animeYear != 0 && tInfo.Year != 0 {
				diff := animeYear - tInfo.Year
				if max(diff, -diff) > tYearThreshold {
					continue
				}
			}

			matchedAnimeSeason := canBeAnimeSeason && slices.Contains(tInfo.Seasons, animeSeason)

			absTvdbMap := anidbGroup.TVDBEpisodeMaps.GetAbsoluteOrderSeasonMap()
			animeTMapIdx, absoluteTMapIdx := -1, -1

			for _, tvdbMap := range anidbGroup.TVDBEpisodeMaps {
				isAnimeSeason := matchedAnimeSeason && tvdbMap.TVDBSeason != 0 && tvdbMap.TVDBSeason != -1
				isTVSeason := tvdbMap.AniDBSeason > 0 && tvdbMap.TVDBSeason > 0 && slices.Contains(tInfo.Seasons, tvdbMap.TVDBSeason)

				if hasEpisodes {
					isAnimeSeason = isAnimeSeason && tSeasonCount == 1
					if isAnimeSeason {
						if tEpisodesAreAbsolute {
							tvdbMap := anidbGroup.TVDBEpisodeMaps.GetAbsoluteOrderSeasonMap()
							tvAbsEpiStart, tvAbsEpiEnd := tvdbMap.TVDBEpisodeBoundary()
							isAnimeSeason = tvAbsEpiStart <= tFirstEpisode && tLastEpisode <= tvAbsEpiEnd
						} else {
							aniEpiStart, aniEpiEnd := tvdbMap.AniDBEpisodeBoundary()
							isAnimeSeason = aniEpiStart == tFirstEpisode && tLastEpisode == aniEpiEnd
						}
					}

					isTVSeason = isTVSeason && tSeasonCount == 1
					if isTVSeason {
						if tEpisodesAreAbsolute {
							tvdbMap := anidbGroup.TVDBEpisodeMaps.GetAbsoluteOrderSeasonMap()
							tvEpiStart, tvEpiEnd := tvdbMap.TVDBEpisodeBoundary()
							isTVSeason = tvEpiStart <= tFirstEpisode && tLastEpisode <= tvEpiEnd
						} else {
							if tvdbMap.HasEpisodeRange() {
								tvEpiStart, tvEpiEnd := tvdbMap.TVDBEpisodeBoundary()
								if tEpisodeCount == 1 {
									isTVSeason = tvEpiStart <= tFirstEpisode && tLastEpisode <= tvEpiEnd
								} else if tEpisodeCount > 1 {
									isTVSeason = tvEpiStart == tFirstEpisode && tLastEpisode == tvEpiEnd
								}
							}
						}
					}
				}

				if isTVSeason {
					if _, seen := idSeen[anidbGroup.AniDBId]; !seen {
						idSeen[anidbGroup.AniDBId] = struct{}{}

						// for absolute and anime season

						var absoluteTMap *torrentMap

						animeTMap := torrentMap{
							anidbId:    anidbGroup.AniDBId,
							seasonType: anidb.TorrentSeasonTypeTypeAnime,
							season:     animeSeason,
						}

						if hasAbsoluteOrder {
							matched := absTvdbMap != nil
							start, end := -1, -1
							if matched {
								absStart, absEnd := absTvdbMap.TVDBEpisodeBoundary()
								if hasEpisodes {
									if tEpisodesAreAbsolute {
										firstEpi, lastEpi := tFirstEpisode, tLastEpisode
										start = max(absStart, firstEpi)
										end = min(absEnd, lastEpi)
										if end == 0 {
											end = lastEpi
										}
										matched = start <= end
									} else {
										firstEpi, lastEpi := anidbGroup.TVDBEpisodeMaps.ToTVDBAbsoluteRange(tInfo.Seasons[0], tFirstEpisode, tLastEpisode)
										start = max(absStart, firstEpi)
										end = min(absEnd, lastEpi)
										if end == 0 {
											end = lastEpi
										}
										matched = start <= end
									}
								} else {
									start, end = absStart, absEnd
								}
							}
							if matched {
								absoluteTMap = &torrentMap{
									anidbId:    anidbGroup.AniDBId,
									seasonType: anidb.TorrentSeasonTypeAbsolute,
									season:     absTvdbMap.TVDBSeason,
								}

								if start > 0 && end > 0 {
									absoluteTMap.episodeStart, absoluteTMap.episodeEnd = start, end
									animeTMap.episodeStart, animeTMap.episodeEnd = start-absTvdbMap.Offset, end-absTvdbMap.Offset
								}
							}
						} else if hasEpisodes {
							if !anidbGroup.TVDBEpisodeMaps.HasSplitedTVSeasons() {
								animeTMap.episodeStart, animeTMap.episodeEnd = tFirstEpisode, tLastEpisode
							}
						}

						if animeTMap.season == -1 && tvdbMap.AniDBSeason == 1 && tvdbMap.TVDBSeason == 1 {
							animeTMap.season = 1
						}

						if absoluteTMap != nil {
							tMaps = append(tMaps, *absoluteTMap)
							absoluteTMapIdx = len(tMaps) - 1
						}
						if animeTMap.season != -1 {
							tMaps = append(tMaps, animeTMap)
							animeTMapIdx = len(tMaps) - 1
						}
					}
					tMap := torrentMap{
						anidbId:    anidbGroup.AniDBId,
						seasonType: anidb.TorrentSeasonTypeTV,
						season:     tvdbMap.TVDBSeason,
					}
					if hasEpisodes {
						tMap.episodeStart, tMap.episodeEnd = tFirstEpisode, tLastEpisode
						if tEpisodesAreAbsolute {
							tMap.episodeStart -= absTvdbMap.Offset
							tMap.episodeEnd -= absTvdbMap.Offset
						}
					} else {
						if tvdbMap.Start != 0 && tvdbMap.End != 0 {
							tMap.episodeStart, tMap.episodeEnd = tvdbMap.TVDBEpisodeBoundary()
						}

						if absoluteTMapIdx != -1 {
							absoluteTMap := &tMaps[absoluteTMapIdx]
							if absoluteTMap.episodeStart == 0 {
								absoluteTMap.episodeStart = tvdbMap.Start + absTvdbMap.Offset
							}
							absoluteTMap.episodeEnd = tvdbMap.End + absTvdbMap.Offset
						}
						if animeTMapIdx != -1 {
							animeTMap := &tMaps[animeTMapIdx]
							if animeTMap.episodeStart == 0 {
								animeTMap.episodeStart = tvdbMap.Start
							}
							animeTMap.episodeEnd = tvdbMap.End
						}
					}

					tMaps = append(tMaps, tMap)
				} else if isAnimeSeason {
					if _, seen := idSeen[anidbGroup.AniDBId]; !seen {
						idSeen[anidbGroup.AniDBId] = struct{}{}
						animeTMap := torrentMap{
							anidbId:    anidbGroup.AniDBId,
							seasonType: anidb.TorrentSeasonTypeTypeAnime,
							season:     animeSeason,
						}

						var absoluteTMap *torrentMap
						if hasAbsoluteOrder {
							tvdbMap := anidbGroup.TVDBEpisodeMaps.GetAbsoluteOrderSeasonMap()
							if tvdbMap != nil {
								absoluteTMap = &torrentMap{
									anidbId:    anidbGroup.AniDBId,
									seasonType: anidb.TorrentSeasonTypeAbsolute,
									season:     tvdbMap.TVDBSeason,
								}
								absoluteTMap.episodeStart, absoluteTMap.episodeEnd = tvdbMap.Start+tvdbMap.Offset, tvdbMap.End+tvdbMap.Offset

								animeTMap.episodeStart, animeTMap.episodeEnd = tvdbMap.Start, tvdbMap.End
							}
						}

						if absoluteTMap != nil {
							tMaps = append(tMaps, *absoluteTMap)
						}
						tMaps = append(tMaps, animeTMap)
					}
					if tvdbMap.TVDBSeason != -1 {
						tMap := torrentMap{
							anidbId:    anidbGroup.AniDBId,
							seasonType: anidb.TorrentSeasonTypeTV,
							season:     tvdbMap.TVDBSeason,
						}
						tMap.episodeStart, tMap.episodeEnd = tvdbMap.Start+tvdbMap.Offset, tvdbMap.End+tvdbMap.Offset
						tMaps = append(tMaps, tMap)
					}
				} else if tEpisodesAreAbsolute {

				}
			}
		}
		return tMaps, nil
	}

	if tEpisodeCount != 0 {
		hasAbsoluteOrder := tvdbMaps.HasAbsoluteOrder()

		if hasAbsoluteOrder {
			tFirstEpisode := tInfo.Episodes[0]
			tLastEpisode := tInfo.Episodes[len(tInfo.Episodes)-1]

			for _, anidbGroup := range tvdbMaps.GroupByAniDBId() {
				if !anidbGroup.TVDBEpisodeMaps.HasAbsoluteOrder() {
					continue
				}

				animeYear := titles.GetYear(anidbGroup.AniDBId)
				if animeYear != 0 && tInfo.Year != 0 {
					diff := animeYear - tInfo.Year
					if max(diff, -diff) > tYearThreshold {
						continue
					}
				}

				aniStart, aniEnd := -1, -1
				for i := range anidbGroup.TVDBEpisodeMaps {
					tvdbMap := &anidbGroup.TVDBEpisodeMaps[i]
					if tvdbMap.HasAbsoluteOrder() {
						absStart, absEnd := tvdbMap.TVDBEpisodeBoundary()
						start, end := max(absStart, tFirstEpisode), min(absEnd, tLastEpisode)
						if end == 0 {
							end = tLastEpisode
						}
						if start <= end {
							tMap := torrentMap{
								anidbId:    anidbGroup.AniDBId,
								seasonType: anidb.TorrentSeasonTypeAbsolute,
								season:     tvdbMap.TVDBSeason,
							}
							tMap.episodeStart = start
							tMap.episodeEnd = end
							tMaps = append(tMaps, tMap)

							aniStart, aniEnd = start-tvdbMap.Offset, end-tvdbMap.Offset
							animeTMap := torrentMap{
								anidbId:      anidbGroup.AniDBId,
								seasonType:   anidb.TorrentSeasonTypeTypeAnime,
								season:       titles.GetSeason(anidbGroup.AniDBId),
								episodeStart: aniStart,
								episodeEnd:   aniEnd,
							}
							tMaps = append(tMaps, animeTMap)
							tFirstEpisode = end + tvdbMap.Offset + 1
						}
					} else if 0 < aniStart && aniStart <= aniEnd {
						start, end := max(aniStart, tvdbMap.Start), min(aniEnd, tvdbMap.End)
						if start <= end {
							tMap := torrentMap{
								anidbId:      anidbGroup.AniDBId,
								seasonType:   anidb.TorrentSeasonTypeTV,
								season:       tvdbMap.TVDBSeason,
								episodeStart: start + tvdbMap.Offset,
								episodeEnd:   end + tvdbMap.Offset,
							}
							tMaps = append(tMaps, tMap)
							aniStart = end + 1
						} else if aniStart == tvdbMap.Start && tvdbMap.End == 0 {
							tMap := torrentMap{
								anidbId:    anidbGroup.AniDBId,
								seasonType: anidb.TorrentSeasonTypeTV,
								season:     tvdbMap.TVDBSeason,
							}
							tMap.episodeStart, tMap.episodeEnd = aniStart+tvdbMap.Offset, aniEnd+tvdbMap.Offset
							tMaps = append(tMaps, tMap)
						}
					}
				}
			}

			return tMaps, nil
		} else {
			for _, tvdbMap := range tvdbMaps {
				if tvdbMap.TVDBSeason == 1 {
					animeYear := titles.GetYear(tvdbMap.AniDBId)
					if animeYear != 0 && tInfo.Year != 0 {
						diff := (animeYear - tInfo.Year)
						if max(diff, -diff) > tYearThreshold {
							continue
						}
					}

					animeSeason := titles.GetSeason(tvdbMap.AniDBId)
					if animeSeason != -1 {
						tMap := torrentMap{
							anidbId:    tvdbMap.AniDBId,
							seasonType: anidb.TorrentSeasonTypeTypeAnime,
							season:     animeSeason,
						}
						tMap.episodeStart, tMap.episodeEnd = tInfo.Episodes[0], tInfo.Episodes[len(tInfo.Episodes)-1]
						tMaps = append(tMaps, tMap)
					}

					tMap := torrentMap{
						anidbId:    tvdbMap.AniDBId,
						seasonType: anidb.TorrentSeasonTypeTV,
						season:     tvdbMap.TVDBSeason,
					}
					tMap.episodeStart, tMap.episodeEnd = tInfo.Episodes[0], tInfo.Episodes[len(tInfo.Episodes)-1]
					tMaps = append(tMaps, tMap)
				}
			}
			return tMaps, nil
		}
	}

	return tMaps, nil
}

func InitMapAniDBTorrentWorker(conf *WorkerConfig) *Worker {
	if !config.Feature.IsEnabled("anime") {
		return nil
	}

	log := logger.Scoped("worker/map_anidb_torrent")

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	isRunning := false
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(30 * time.Minute),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else {
					isRunning = false
				}
				worker.onEnd()
			}()

			for {
				wait, reason := worker.shouldWait()
				if !wait {
					break
				}
				log.Info("waiting, " + reason)
				time.Sleep(5 * time.Minute)
			}
			worker.onStart()

			if isRunning {
				return nil
			}

			isRunning = true

			if !isAnidbTitlesSynced() {
				log.Info("AniDB titles not synced yet today, skipping")
				return nil
			}

			if !isAniDBTVDBEpisodeMapSynced() {
				log.Info("AniDB TVDB episode maps not synced yet today, skipping")
				return nil
			}

			if !isAnimeAPISynced() {
				log.Info("AnimeAPI not synced yet today, skipping")
				return nil
			}

			if !isManamiAnimeDatabaseSynced() {
				log.Info("Manami anime database not synced yet today, skipping")
				return nil
			}

			batch_size := 10000
			chunk_size := 1000
			if db.Dialect == db.DBDialectPostgres {
				batch_size = 20000
				chunk_size = 2000
			}

			totalCount := 0
			for {
				hashes, err := torrent_info.GetAniDBUnmappedHashes(batch_size)
				if err != nil {
					return err
				}

				var wg sync.WaitGroup
				for cHashes := range slices.Chunk(hashes, chunk_size) {
					wg.Add(1)
					go func() {
						defer wg.Done()

						items := []anidb.AniDBTorrent{}
						tInfoByHash, err := torrent_info.GetByHashes(cHashes)
						if err != nil {
							log.Error("failed to get torrent info", "error", err)
							return
						}
						for hash, tInfo := range tInfoByHash {
							if !tInfo.IsParsed() {
								continue
							}

							if tInfo.Title == "" {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								continue
							}

							anidbTitleIds, err := anidb.SearchIdsByTitle(tInfo.Title, nil, 0, 1)
							if len(anidbTitleIds) == 0 {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								continue
							}
							anidbId := anidbTitleIds[0]

							tvdbMaps, err := anidb.GetTVDBEpisodeMaps(anidbId)
							if err != nil {
								log.Error("failed to get tvdb episode maps", "error", err, "anidb_id", anidbId)
								continue
							}

							if len(tvdbMaps) == 0 {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								continue
							}

							anidbTitles, err := tvdbMaps.GetAniDBTitles()
							if err != nil {
								log.Error("failed to get anidb titles from tvdb episode maps", "error", err, "anidb_id", anidbId, "tvdb_id", tvdbMaps.GetTVDBId())
								continue
							}

							if anidbTitles[0].Type == "movie" && (len(tInfo.Seasons) > 0 || len(tInfo.Episodes) > 0) {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								continue
							}

							titleMatchRatio := 0
							for i := range anidbTitles {
								title := &anidbTitles[i]
								titleMatchRatio = max(titleMatchRatio, fuzzy.UQRatio(tInfo.Title, title.Value))
								if titleMatchRatio >= 85 {
									break
								}
							}
							if titleMatchRatio < 85 {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
								log.Debug("title match ratio is low", "anidb_id", anidbId, "tvdb_id", tvdbMaps.GetTVDBId(), "t_title", tInfo.Title, "ratio", titleMatchRatio)
								continue
							}

							torrentMaps, err := prepareAniDBTorrentMaps(tvdbMaps, anidbTitles, tInfo)
							if err != nil {
								log.Error("failed to match anidb ids in tvdb episode maps", "error", err, "anidb_id", anidbId, "tvdb_id", tvdbMaps.GetTVDBId())
								continue
							}

							if len(torrentMaps) == 0 {
								items = append(items, anidb.AniDBTorrent{
									Hash: hash,
								})
							} else {
								for _, tMap := range torrentMaps {
									tor := anidb.AniDBTorrent{
										TId:          tMap.anidbId,
										Hash:         hash,
										SeasonType:   tMap.seasonType,
										Season:       tMap.season,
										EpisodeStart: tMap.episodeStart,
										EpisodeEnd:   tMap.episodeEnd,
										Episodes:     tMap.episodes,
									}
									items = append(items, tor)
								}
							}
						}

						if err := anidb.UpsertTorrents(items); err != nil {
							log.Error("failed to map anidb torrent", "error", err)
							return
						}

						log.Info("mapped anidb torrent", "count", len(items))
					}()
				}
				wg.Wait()

				count := len(hashes)
				totalCount += count
				log.Info("processed torrents", "totalCount", totalCount)

				if count < batch_size {
					break
				}

				time.Sleep(200 * time.Millisecond)
			}

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			isRunning = false
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	if task, err := worker.scheduler.Lookup(id); err == nil && task != nil {
		t := task.Clone()
		t.Interval = 90 * time.Second
		t.RunOnce = true
		worker.scheduler.Add(t)
	}

	return worker
}
