package tvdb

import (
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
)

var tvdbItemPool = pond.NewResultPool[*TVDBItem](10)

func GetIMDBIdsForTVDBIds(tvdbMovieIds, tvdbSeriesIds []string) (map[string]string, map[string]string, error) {
	movieImdbIdByTvdbId, seriesImdbIdByTvdbId, err := imdb_title.GetIMDBIdByTVDBId(tvdbMovieIds, tvdbSeriesIds)
	if err != nil {
		return nil, nil, err
	}

	var missingTVDBMovieIds, missingTVDBSeriesIds []string
	if len(movieImdbIdByTvdbId) < len(tvdbMovieIds) {
		missingTVDBMovieIds = make([]string, 0, len(tvdbMovieIds)-len(movieImdbIdByTvdbId))
		for _, id := range tvdbMovieIds {
			if _, ok := movieImdbIdByTvdbId[id]; !ok {
				missingTVDBMovieIds = append(missingTVDBMovieIds, id)
			}
		}
	}
	if len(seriesImdbIdByTvdbId) < len(tvdbSeriesIds) {
		missingTVDBSeriesIds = make([]string, 0, len(tvdbSeriesIds)-len(seriesImdbIdByTvdbId))
		for _, id := range tvdbSeriesIds {
			if _, ok := seriesImdbIdByTvdbId[id]; !ok {
				missingTVDBSeriesIds = append(missingTVDBSeriesIds, id)
			}
		}
	}

	if len(missingTVDBMovieIds) > 0 || len(missingTVDBSeriesIds) > 0 {
		tvdbClient := GetAPIClient()

		log.Debug("fetching remote ids for tvdb", "movie_count", len(missingTVDBMovieIds), "series_count", len(missingTVDBSeriesIds))

		movieGroup := tvdbItemPool.NewGroup()
		for _, movieId := range missingTVDBMovieIds {
			movieGroup.SubmitErr(func() (*TVDBItem, error) {
				li := TVDBItem{
					Type: TVDBItemTypeMovie,
					Id:   util.SafeParseInt(movieId, -1),
				}
				err := li.Fetch(tvdbClient, true)
				return &li, err
			})
		}

		seriesGroup := tvdbItemPool.NewGroup()
		for _, seriesId := range missingTVDBSeriesIds {
			seriesGroup.SubmitErr(func() (*TVDBItem, error) {
				li := TVDBItem{
					Type: TVDBItemTypeSeries,
					Id:   util.SafeParseInt(seriesId, -1),
				}
				err := li.Fetch(tvdbClient, true)
				return &li, err
			})
		}

		movieItems, err := movieGroup.Wait()
		if err != nil {
			log.Error("failed to fetch movie remote ids from tvdb", "error", err)
		}
		for i, item := range movieItems {
			if item == nil || item.IdMap == nil || item.IdMap.IMDB == "" {
				continue
			}
			tvdbId := missingTVDBMovieIds[i]
			movieImdbIdByTvdbId[tvdbId] = item.IdMap.IMDB
		}

		seriesItems, err := seriesGroup.Wait()
		if err != nil {
			log.Error("failed to fetch series remote ids from tvdb", "error", err)
		}
		for i, item := range seriesItems {
			if item == nil || item.IdMap == nil || item.IdMap.IMDB == "" {
				continue
			}
			tvdbId := missingTVDBSeriesIds[i]
			seriesImdbIdByTvdbId[tvdbId] = item.IdMap.IMDB
		}
	}

	return movieImdbIdByTvdbId, seriesImdbIdByTvdbId, nil
}

var tvdbSearchByRemoteIdPool = pond.NewResultPool[*SearchByRemoteIdData](10)

func GetTVDBIdsForIMDBIds(imdbIds []string) (map[string]string, error) {
	tvdbIdByImdbId, err := imdb_title.GetTVDBIdByIMDBId(imdbIds)
	if err != nil {
		return nil, err
	}

	missingImdbIds := []string{}
	for _, imdbId := range imdbIds {
		if id, ok := tvdbIdByImdbId[imdbId]; !ok || id == "" {
			missingImdbIds = append(missingImdbIds, imdbId)
		}
	}

	if len(missingImdbIds) == 0 {
		return tvdbIdByImdbId, nil
	}

	tvdbClient := GetAPIClient()

	log.Debug("fetching tvdb ids for imdb ids", "count", len(missingImdbIds))

	wg := tvdbSearchByRemoteIdPool.NewGroup()
	for _, imdbId := range missingImdbIds {
		wg.SubmitErr(func() (*SearchByRemoteIdData, error) {
			res, err := tvdbClient.SearchByRemoteId(&SearchByRemoteIdParams{
				RemoteId: imdbId,
			})
			return &res.Data, err
		})
	}

	results, err := wg.Wait()
	if err != nil {
		log.Error("failed to fetch tvdb ids for imdb ids", "error", err)
	}
	newIdMaps := make([]meta.IdMap, 0, len(missingImdbIds))
	for i, result := range results {
		if result == nil {
			continue
		}
		imdbId := missingImdbIds[i]
		if movie := result.Movie; movie != nil {
			tvdbId := strconv.Itoa(movie.Id)
			tvdbIdByImdbId[imdbId] = tvdbId
			newIdMaps = append(newIdMaps, meta.IdMap{
				Type: meta.IdTypeMovie,
				IMDB: imdbId,
				TVDB: tvdbId,
			})
		} else if series := result.Series; series != nil {
			tvdbId := strconv.Itoa(series.Id)
			tvdbIdByImdbId[imdbId] = tvdbId
			newIdMaps = append(newIdMaps, meta.IdMap{
				Type: meta.IdTypeShow,
				IMDB: imdbId,
				TVDB: tvdbId,
			})
		}
	}

	go util.LogError(log, meta.SetIdMaps(newIdMaps, meta.IdProviderIMDB), "failed to set id maps")

	return tvdbIdByImdbId, nil
}
