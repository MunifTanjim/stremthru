package stremio_list

import (
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
)

var tvdbItemPool = pond.NewResultPool[*tvdb.TVDBItem](10)

func getIMDBIdsForTVDBIds(tvdbMovieIds, tvdbSeriesIds []string) (map[string]string, map[string]string, error) {
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
		log.Debug("fetching remote ids for tvdb", "movie_count", len(missingTVDBMovieIds), "series_count", len(missingTVDBSeriesIds))
		tvdbClient := tvdb.GetAPIClient()
		movieGroup := tvdbItemPool.NewGroup()
		for _, movieId := range missingTVDBMovieIds {
			movieGroup.SubmitErr(func() (*tvdb.TVDBItem, error) {
				li := tvdb.TVDBItem{
					Type: tvdb.TVDBItemTypeMovie,
					Id:   util.SafeParseInt(movieId, -1),
				}
				err := li.Fetch(tvdbClient)
				return &li, err
			})
		}
		seriesGroup := tvdbItemPool.NewGroup()
		for _, seriesId := range missingTVDBSeriesIds {
			seriesGroup.SubmitErr(func() (*tvdb.TVDBItem, error) {
				li := tvdb.TVDBItem{
					Type: tvdb.TVDBItemTypeSeries,
					Id:   util.SafeParseInt(seriesId, -1),
				}
				err := li.Fetch(tvdbClient)
				return &li, err
			})
		}

		movieItems, err := movieGroup.Wait()
		if err != nil {
			log.Error("failed to fetch movie remote ids from tvdb", "error", err)
		}
		for i, item := range movieItems {
			if item == nil || item.Ids == nil || item.Ids.IMDBId == "" {
				continue
			}
			tvdbId := missingTVDBMovieIds[i]
			movieImdbIdByTvdbId[tvdbId] = item.Ids.IMDBId
		}
		seriesItems, err := seriesGroup.Wait()
		if err != nil {
			log.Error("failed to fetch series remote ids from tvdb", "error", err)
		}
		for i, item := range seriesItems {
			if item == nil || item.Ids == nil || item.Ids.IMDBId == "" {
				continue
			}
			tvdbId := missingTVDBSeriesIds[i]
			seriesImdbIdByTvdbId[tvdbId] = item.Ids.IMDBId
		}
	}

	return movieImdbIdByTvdbId, seriesImdbIdByTvdbId, nil
}

var tvdbSearchByRemoteIdPool = pond.NewResultPool[*tvdb.SearchByRemoteIdData](10)

func getTVDBIdsForIMDBIds(imdbIds []string) (map[string]string, error) {
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

	tvdbClient := tvdb.GetAPIClient()

	log.Debug("fetching tvdb ids for imdb ids", "count", len(missingImdbIds))

	wg := tvdbSearchByRemoteIdPool.NewGroup()
	for _, imdbId := range missingImdbIds {
		wg.SubmitErr(func() (*tvdb.SearchByRemoteIdData, error) {
			res, err := tvdbClient.SearchByRemoteId(&tvdb.SearchByRemoteIdParams{
				RemoteId: imdbId,
			})
			return &res.Data, err
		})
	}

	results, err := wg.Wait()
	if err != nil {
		log.Error("failed to fetch tvdb ids for imdb ids", "error", err)
	}
	newMappings := make([]imdb_title.BulkRecordMappingInputItem, 0, len(missingImdbIds))
	for i, result := range results {
		if result == nil {
			continue
		}
		imdbId := missingImdbIds[i]
		if movie := result.Movie; movie != nil {
			tvdbId := strconv.Itoa(movie.Id)
			tvdbIdByImdbId[imdbId] = tvdbId
			newMappings = append(newMappings, imdb_title.BulkRecordMappingInputItem{
				IMDBId: imdbId,
				TVDBId: tvdbId,
			})
		} else if series := result.Series; series != nil {
			tvdbId := strconv.Itoa(series.Id)
			tvdbIdByImdbId[imdbId] = tvdbId
			newMappings = append(newMappings, imdb_title.BulkRecordMappingInputItem{
				IMDBId: imdbId,
				TVDBId: tvdbId,
			})
		}
	}

	go imdb_title.BulkRecordMapping(newMappings)

	return tvdbIdByImdbId, nil
}
