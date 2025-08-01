package stremio_list

import (
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/internal/tmdb"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/alitto/pond/v2"
)

type ExtraData struct {
	Skip  int
	Genre string
}

func getExtra(r *http.Request) *ExtraData {
	extra := &ExtraData{}
	if extraParams := GetPathValue(r, "extra"); extraParams != "" {
		if q, err := url.ParseQuery(extraParams); err == nil {
			if skipStr := q.Get("skip"); skipStr != "" {
				if skip, err := strconv.Atoi(skipStr); err == nil {
					extra.Skip = skip
				}
			}
			if genre := q.Get("genre"); genre != "" {
				extra.Genre = genre
			}
		}
	}
	return extra
}

func getIMDBMetaFromMDBList(imdbIds []string, mdblistAPIKey string) (map[string]imdb_title.IMDBTitleMeta, error) {
	byId := map[string]imdb_title.IMDBTitleMeta{}

	metas, err := imdb_title.GetMetasByIds(imdbIds)
	if err != nil {
		return nil, err
	}
	for _, meta := range metas {
		byId[meta.TId] = meta
	}

	staleOrMissingIds := []string{}
	for _, imdbId := range imdbIds {
		if meta, ok := byId[imdbId]; !ok || meta.IsStale() {
			staleOrMissingIds = append(staleOrMissingIds, imdbId)
		}
	}

	staleOrMissingCount := len(staleOrMissingIds)

	if staleOrMissingCount == 0 {
		return byId, nil
	}

	log.Debug("fetching media info from mdblist", "count", staleOrMissingCount)
	params := &mdblist.GetMediaInfoBatchParams{
		MediaProvider: "imdb",
		MediaType:     "any",
		Ids:           staleOrMissingIds,
	}
	params.APIKey = mdblistAPIKey
	newMetas := make([]imdb_title.IMDBTitleMeta, 0, staleOrMissingCount)
	newMappings := make([]imdb_title.BulkRecordMappingInputItem, 0, staleOrMissingCount)
	res, err := mdblistClient.GetMediaInfoBatch(params)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range res.Data {
		mInfo := &res.Data[i]
		meta := imdb_title.IMDBTitleMeta{
			TId:         mInfo.Ids.IMDB,
			Description: mInfo.Description,
			Runtime:     mInfo.Runtime,
			Poster:      mInfo.Poster,
			Backdrop:    mInfo.Backdrop,
			Trailer:     mInfo.Trailer,
			Rating:      mInfo.Score,
			MPARating:   mInfo.Certification,
			UpdatedAt:   db.Timestamp{Time: now},
			Genres:      make([]string, len(mInfo.Genres)),
		}
		for i := range mInfo.Genres {
			meta.Genres[i] = mInfo.Genres[i].Title
		}
		newMetas = append(newMetas, meta)
		newMappings = append(newMappings, imdb_title.BulkRecordMappingInputItem{
			IMDBId:  mInfo.Ids.IMDB,
			TMDBId:  strconv.Itoa(mInfo.Ids.TMDB),
			TVDBId:  strconv.Itoa(mInfo.Ids.TVDB),
			TraktId: strconv.Itoa(mInfo.Ids.Trakt),
			MALId:   strconv.Itoa(mInfo.Ids.MAL),
		})
		byId[meta.TId] = meta
	}

	go imdb_title.BulkRecordMapping(newMappings)
	if err = imdb_title.UpsertMetas(newMetas); err != nil {
		return nil, err
	}
	return byId, nil
}

var tmdbMovieExternalIdsPool = pond.NewResultPool[*tmdb.GetMovieExternalIdsData](10)
var tmdbShowExternalIdsPool = pond.NewResultPool[*tmdb.GetTVExternalIdsData](10)

func getIMDBIdsForTMDBIds(tokenId string, tmdbMovieIds, tmdbShowIds []string) (map[string]string, map[string]string, error) {
	movieImdbIdByTmdbId, showImdbIdByTmdbId, err := imdb_title.GetIMDBIdByTMDBId(tmdbMovieIds, tmdbShowIds)
	if err != nil {
		return nil, nil, err
	}

	var missingTMDBMovieIds, missingTMDBShowIds []string
	if len(movieImdbIdByTmdbId) < len(tmdbMovieIds) {
		missingTMDBMovieIds = make([]string, 0, len(tmdbMovieIds)-len(movieImdbIdByTmdbId))
		for _, id := range tmdbMovieIds {
			if _, ok := movieImdbIdByTmdbId[id]; !ok {
				missingTMDBMovieIds = append(missingTMDBMovieIds, id)
			}
		}
	}
	if len(showImdbIdByTmdbId) < len(tmdbShowIds) {
		missingTMDBShowIds = make([]string, 0, len(tmdbShowIds)-len(showImdbIdByTmdbId))
		for _, id := range tmdbShowIds {
			if _, ok := showImdbIdByTmdbId[id]; !ok {
				missingTMDBShowIds = append(missingTMDBShowIds, id)
			}
		}
	}

	if len(missingTMDBMovieIds) > 0 || len(missingTMDBShowIds) > 0 {
		log.Debug("fetching external ids for tmdb", "movie_count", len(missingTMDBMovieIds), "show_count", len(missingTMDBShowIds))
		tmdbClient := tmdb.GetAPIClient(tokenId)
		movieGroup := tmdbMovieExternalIdsPool.NewGroup()
		for _, movieId := range missingTMDBMovieIds {
			movieGroup.SubmitErr(func() (*tmdb.GetMovieExternalIdsData, error) {
				res, err := tmdbClient.GetMovieExternalIds(&tmdb.GetMovieExternalIdsParams{
					MovieId: movieId,
				})
				return &res.Data, err
			})
		}
		showGroup := tmdbShowExternalIdsPool.NewGroup()
		for _, showId := range missingTMDBShowIds {
			showGroup.SubmitErr(func() (*tmdb.GetTVExternalIdsData, error) {
				res, err := tmdbClient.GetTVExternalIds(&tmdb.GetTVExternalIdsParams{
					SeriesId: showId,
				})
				return &res.Data, err
			})
		}

		bulkRecordMappingItems := make([]imdb_title.BulkRecordMappingInputItem, 0, len(missingTMDBMovieIds)+len(missingTMDBShowIds))

		movieExternalIds, err := movieGroup.Wait()
		if err != nil {
			log.Error("failed to fetch movie external ids from tmdb", "error", err)
		}
		for i, movieExternalId := range movieExternalIds {
			if movieExternalId == nil || movieExternalId.IMDBId == "" {
				continue
			}
			tmdbId := missingTMDBMovieIds[i]
			movieImdbIdByTmdbId[tmdbId] = movieExternalId.IMDBId
			bulkRecordMappingItems = append(bulkRecordMappingItems, imdb_title.BulkRecordMappingInputItem{
				IMDBId: movieExternalId.IMDBId,
				TMDBId: tmdbId,
			})
		}
		showExternalIds, err := showGroup.Wait()
		if err != nil {
			log.Error("failed to fetch show external ids from tmdb", "error", err)
		}
		for i, showExternalId := range showExternalIds {
			if showExternalId == nil || showExternalId.IMDBId == "" {
				continue
			}
			tmdbId := missingTMDBShowIds[i]
			showImdbIdByTmdbId[tmdbId] = showExternalId.IMDBId
			bulkRecordMappingItems = append(bulkRecordMappingItems, imdb_title.BulkRecordMappingInputItem{
				IMDBId: showExternalId.IMDBId,
				TMDBId: tmdbId,
				TVDBId: strconv.Itoa(showExternalId.TVDBId),
			})
		}

		go imdb_title.BulkRecordMapping(bulkRecordMappingItems)
	}

	return movieImdbIdByTmdbId, showImdbIdByTmdbId, nil
}

var tmdbFindByIdPool = pond.NewResultPool[*tmdb.FindByIdData](10)

func getTMDBIdsForIMDBIds(tokenId string, imdbIds []string) (map[string]string, error) {
	tmdbIdByImdbId, err := imdb_title.GetTMDBIdByIMDBId(imdbIds)
	if err != nil {
		return nil, err
	}

	missingImdbIds := []string{}
	for _, imdbId := range imdbIds {
		if id, ok := tmdbIdByImdbId[imdbId]; !ok || id == "" {
			missingImdbIds = append(missingImdbIds, imdbId)
		}
	}

	if len(missingImdbIds) == 0 {
		return tmdbIdByImdbId, nil
	}

	tmdbClient := tmdb.GetAPIClient(tokenId)

	log.Debug("fetching tmdb ids for imdb ids", "count", len(missingImdbIds))

	tmdbGroup := tmdbFindByIdPool.NewGroup()
	for _, imdbId := range missingImdbIds {
		tmdbGroup.SubmitErr(func() (*tmdb.FindByIdData, error) {
			res, err := tmdbClient.FindById(&tmdb.FindByIdParams{
				ExternalSource: "imdb_id",
				ExternalId:     imdbId,
			})
			return &res.Data, err
		})
	}

	results, err := tmdbGroup.Wait()
	if err != nil {
		log.Error("failed to fetch tmdb ids for imdb ids", "error", err)
	}
	newMappings := make([]imdb_title.BulkRecordMappingInputItem, 0, len(missingImdbIds))
	tmdbItems := make([]tmdb.TMDBItem, 0, len(missingImdbIds))
	for i, result := range results {
		if result == nil {
			continue
		}
		imdbId := missingImdbIds[i]
		if movie := result.Movie(); movie != nil {
			tmdbId := strconv.Itoa(movie.Id)
			tmdbIdByImdbId[imdbId] = tmdbId
			newMappings = append(newMappings, imdb_title.BulkRecordMappingInputItem{
				IMDBId: imdbId,
				TMDBId: tmdbId,
			})
			tmdbItems = append(tmdbItems, tmdb.TMDBItem{
				Id:            movie.Id,
				Type:          tmdb.MediaTypeMovie,
				IsPartial:     true,
				Title:         movie.Title,
				OriginalTitle: movie.OriginalTitle,
				Overview:      movie.Overview,
				ReleaseDate:   db.DateOnly{Time: movie.GetReleaseDate()},
				IsAdult:       movie.Adult,
				Backdrop:      movie.BackdropPath,
				Poster:        movie.PosterPath,
				Popularity:    movie.Popularity,
				VoteAverage:   movie.VoteAverage,
				VoteCount:     movie.VoteCount,
				Genres:        movie.GenreIds,
			})
		} else if show := result.Show(); show != nil {
			tmdbId := strconv.Itoa(show.Id)
			tmdbIdByImdbId[imdbId] = tmdbId
			newMappings = append(newMappings, imdb_title.BulkRecordMappingInputItem{
				IMDBId: imdbId,
				TMDBId: tmdbId,
			})
			tmdbItems = append(tmdbItems, tmdb.TMDBItem{
				Id:            show.Id,
				Type:          tmdb.MediaTypeTVShow,
				IsPartial:     true,
				Title:         show.Name,
				OriginalTitle: show.OriginalName,
				Overview:      show.Overview,
				ReleaseDate:   db.DateOnly{Time: show.GetFirstAirDate()},
				IsAdult:       show.Adult,
				Backdrop:      show.BackdropPath,
				Poster:        show.PosterPath,
				Popularity:    show.Popularity,
				VoteAverage:   show.VoteAverage,
				VoteCount:     show.VoteCount,
				Genres:        show.GenreIds,
			})
		}
	}

	go imdb_title.BulkRecordMapping(newMappings)
	go func() {
		err := tmdb.UpsertItems(db.GetDB(), tmdbItems)
		if err != nil {
			log.Error("failed to upsert tmdb items", "error", err)
		}
	}()

	return tmdbIdByImdbId, nil
}

type catalogItem struct {
	stremio.MetaPreview
	item any
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r, false)
	if err != nil {
		SendError(w, r, err)
		return
	}

	catalogId := GetPathValue(r, "id")

	service, id := parseCatalogId(catalogId)

	rpdbPosterBaseUrl := ""
	if ud.RPDBAPIKey != "" {
		rpdbPosterBaseUrl = "https://api.ratingposterdb.com/" + ud.RPDBAPIKey + "/imdb/poster-default/"
	}

	catalogItems := []catalogItem{}
	switch service {
	case "anilist":
		list := anilist.AniListList{Id: id}
		if err := ud.FetchAniListList(&list, false); err != nil {
			SendError(w, r, err)
			return
		}

		for i := range list.Medias {
			media := &list.Medias[i]

			meta := stremio.MetaPreview{
				Type:        "anime",
				Name:        media.Title,
				Description: media.Description,
				Poster:      media.Cover,
				Background:  media.Banner,
				PosterShape: stremio.MetaPosterShapePoster,
				Genres:      media.Genres,
				ReleaseInfo: strconv.Itoa(media.StartYear),
			}
			catalogItems = append(catalogItems, catalogItem{meta, *media})
		}

	case "mdblist":
		list := mdblist.MDBListList{Id: id}
		if err := ud.FetchMDBListList(&list); err != nil {
			SendError(w, r, err)
			return
		}

		for i := range list.Items {
			item := &list.Items[i]

			poster := item.Poster
			if rpdbPosterBaseUrl != "" {
				poster = rpdbPosterBaseUrl + item.IMDBId + ".jpg?fallback=true"
			}

			meta := stremio.MetaPreview{
				Id:          item.IMDBId,
				Type:        mdblistMediaTypeToResourceType(item.Mediatype, "other"),
				Name:        item.Title,
				Poster:      poster,
				PosterShape: stremio.MetaPosterShapePoster,
				Background:  stremio_shared.GetCinemetaBackgroundURL(item.IMDBId),
				Genres:      item.Genre,
				ReleaseInfo: strconv.Itoa(item.ReleaseYear),
			}
			catalogItems = append(catalogItems, catalogItem{meta, item})
		}

	case "tmdb":
		list := tmdb.TMDBList{Id: id}
		if err := ud.FetchTMDBList(&list); err != nil {
			SendError(w, r, err)
			return
		}

		for i := range list.Items {
			item := &list.Items[i]
			meta := stremio.MetaPreview{
				Name:        item.Title,
				Description: item.Overview,
				Poster:      item.PosterURL(tmdb.PosterSizeW500),
				PosterShape: stremio.MetaPosterShapePoster,
				Background:  item.BackdropURL(tmdb.BackdropSizeW1280),
				Genres:      item.GenreNames(),
				ReleaseInfo: item.ReleaseDate.Format("2006"),
			}
			switch item.Type {
			case tmdb.MediaTypeMovie:
				meta.Type = stremio.ContentTypeMovie
				if ud.MetaIdMovie == "tmdb" {
					meta.Id = "tmdb:" + strconv.Itoa(item.Id)
				}
			case tmdb.MediaTypeTVShow:
				meta.Type = stremio.ContentTypeSeries
				if ud.MetaIdSeries == "tmdb" {
					meta.Id = "tmdb:" + strconv.Itoa(item.Id)
				}
			default:
				continue
			}
			catalogItems = append(catalogItems, catalogItem{meta, item})
		}

	case "trakt":
		list := trakt.TraktList{Id: id}
		if err := ud.FetchTraktList(&list); err != nil {
			SendError(w, r, err)
			return
		}

		for i := range list.Items {
			item := &list.Items[i]
			meta := stremio.MetaPreview{
				Name:        item.Title,
				Description: item.Overview,
				Poster:      item.Poster,
				PosterShape: stremio.MetaPosterShapePoster,
				Background:  item.Fanart,
				Genres:      item.Genres,
				ReleaseInfo: strconv.Itoa(item.Year),
				IMDBRating:  strconv.FormatFloat(float64(item.Rating)/10, 'f', 1, 32),
			}
			switch item.Type {
			case trakt.ItemTypeMovie:
				meta.Type = stremio.ContentTypeMovie
			case trakt.ItemTypeShow:
				meta.Type = stremio.ContentTypeSeries
			default:
				continue
			}
			if meta.Poster != "" {
				meta.Poster = "https://" + meta.Poster
			}
			if meta.Background != "" {
				meta.Background = "https://" + meta.Background
			}
			if item.Trailer != "" {
				if trailer, err := url.Parse(item.Trailer); err == nil && trailer.Host == "youtube.com" {
					meta.Trailers = append(meta.Trailers, stremio.MetaTrailer{
						Source: trailer.Query().Get("v"),
						Type:   "Trailer",
					})
				}
			}
			catalogItems = append(catalogItems, catalogItem{meta, item})
		}

	default:
		shared.ErrorBadRequest(r, "invalid id").Send(w, r)
		return
	}

	extra := getExtra(r)

	if extra.Genre != "" {
		filteredItems := []catalogItem{}
		for i := range catalogItems {
			item := &catalogItems[i]
			if slices.Contains(item.Genres, extra.Genre) {
				filteredItems = append(filteredItems, *item)
			}
		}
		catalogItems = filteredItems
	}

	limit := 100
	totalItems := len(catalogItems)
	catalogItems = catalogItems[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	items := []stremio.MetaPreview{}

	switch service {
	case "anilist":
		medias := make([]anilist.AniListMedia, len(catalogItems))
		for i := range catalogItems {
			item := &catalogItems[i]
			medias[i] = item.item.(anilist.AniListMedia)
		}
		if err := anilist.EnsureIdMap(medias, id); err != nil {
			SendError(w, r, err)
			return
		}

		for i := range catalogItems {
			item := &catalogItems[i]
			media := medias[i]

			if media.IdMap == nil {
				continue
			}

			switch ud.MetaIdAnime {
			case "mal":
				if media.IdMap.MAL != "" {
					item.Id = "mal:" + media.IdMap.MAL
				}
			case "anilist":
				if media.IdMap.AniList != "" {
					item.Id = "anilist:" + media.IdMap.AniList
				}
			case "anidb":
				if media.IdMap.AniDB != "" {
					item.Id = "anidb:" + media.IdMap.AniDB
				}
			}
			if item.Id == "" && media.IdMap.Kitsu != "" {
				item.Id = "kitsu:" + media.IdMap.Kitsu
			}
			if item.Id == "" {
				continue
			}

			if rpdbPosterBaseUrl != "" && media.IdMap.IMDB != "" {
				item.Poster = rpdbPosterBaseUrl + media.IdMap.IMDB + ".jpg?fallback=true"
			}

			items = append(items, item.MetaPreview)
		}

	case "mdblist":
		imdbIds := []string{}
		for i := range catalogItems {
			id := catalogItems[i].Id
			if strings.HasPrefix(id, "tt") {
				imdbIds = append(imdbIds, id)
			}
		}

		metaById, err := getIMDBMetaFromMDBList(imdbIds, ud.MDBListAPIkey)
		if err != nil {
			SendError(w, r, err)
			return
		}

		for i := range catalogItems {
			item := &catalogItems[i]
			if m, ok := metaById[item.Id]; ok {
				item.Description = m.Description
				item.IMDBRating = strconv.FormatFloat(float64(m.Rating)/10, 'f', 1, 32)
				if trailer, err := url.Parse(m.Trailer); err == nil && trailer.Host == "youtube.com" {
					item.Trailers = append(item.Trailers, stremio.MetaTrailer{
						Source: trailer.Query().Get("v"),
						Type:   "Trailer",
					})
				}
			}
			items = append(items, item.MetaPreview)
		}

	case "tmdb":
		tmdbMovieIds := make([]string, 0, len(catalogItems))
		tmdbShowIds := make([]string, 0, len(catalogItems))
		for i := range catalogItems {
			item := catalogItems[i].item.(*tmdb.TMDBItem)
			switch item.Type {
			case tmdb.MediaTypeMovie:
				tmdbMovieIds = append(tmdbMovieIds, strconv.Itoa(item.Id))
			case tmdb.MediaTypeTVShow:
				tmdbShowIds = append(tmdbShowIds, strconv.Itoa(item.Id))
			}
		}

		movieImdbIdByTmdbId, showImdbIdByTmdbId, err := getIMDBIdsForTMDBIds(ud.TMDBTokenId, tmdbMovieIds, tmdbShowIds)
		if err != nil {
			SendError(w, r, err)
			return
		}

		for i := range catalogItems {
			item := &catalogItems[i]
			titem := item.item.(*tmdb.TMDBItem)
			imdbId := ""
			switch titem.Type {
			case tmdb.MediaTypeMovie:
				if id, ok := movieImdbIdByTmdbId[strconv.Itoa(titem.Id)]; ok {
					imdbId = id
				}
			case tmdb.MediaTypeTVShow:
				if id, ok := showImdbIdByTmdbId[strconv.Itoa(titem.Id)]; ok {
					imdbId = id
				}
			}
			if imdbId == "" && item.MetaPreview.Id == "" {
				continue
			}

			if item.MetaPreview.Id == "" {
				item.MetaPreview.Id = imdbId
			}
			if rpdbPosterBaseUrl != "" {
				item.MetaPreview.Poster = rpdbPosterBaseUrl + imdbId + ".jpg?fallback=true"
			}

			items = append(items, item.MetaPreview)
		}

	case "trakt":
		traktMovieIds := make([]string, 0, len(catalogItems))
		traktShowIds := make([]string, 0, len(catalogItems))
		for i := range catalogItems {
			item := catalogItems[i].item.(*trakt.TraktItem)
			switch item.Type {
			case trakt.ItemTypeMovie:
				traktMovieIds = append(traktMovieIds, strconv.Itoa(item.Id))
			case trakt.ItemTypeShow:
				traktShowIds = append(traktShowIds, strconv.Itoa(item.Id))
			}
		}

		movieImdbIdByTraktId, showImdbIdByTraktId, err := imdb_title.GetIMDBIdByTraktId(traktMovieIds, traktShowIds)
		if err != nil {
			SendError(w, r, err)
			return
		}

		for i := range catalogItems {
			item := &catalogItems[i]
			titem := item.item.(*trakt.TraktItem)
			imdbId := ""
			switch titem.Type {
			case trakt.ItemTypeMovie:
				if id, ok := movieImdbIdByTraktId[strconv.Itoa(titem.Id)]; ok {
					imdbId = id
				}
			case trakt.ItemTypeShow:
				if id, ok := showImdbIdByTraktId[strconv.Itoa(titem.Id)]; ok {
					imdbId = id
				}
			}
			if imdbId == "" {
				continue
			}

			item.MetaPreview.Id = imdbId
			if rpdbPosterBaseUrl != "" {
				item.MetaPreview.Poster = rpdbPosterBaseUrl + imdbId + ".jpg?fallback=true"
			}

			items = append(items, item.MetaPreview)
		}
	}

	imdbIdsToFindTmdbIds := []string{}
	if ud.MetaIdMovie != "" || ud.MetaIdSeries != "" {
		for _, item := range items {
			if strings.HasPrefix(item.Id, "tt") {
				switch item.Type {
				case stremio.ContentTypeMovie:
					if ud.MetaIdMovie == "tmdb" {
						imdbIdsToFindTmdbIds = append(imdbIdsToFindTmdbIds, item.Id)
					}
				case stremio.ContentTypeSeries:
					if ud.MetaIdSeries == "tmdb" {
						imdbIdsToFindTmdbIds = append(imdbIdsToFindTmdbIds, item.Id)
					}
				}
			}
		}
	}
	if len(imdbIdsToFindTmdbIds) > 0 {
		tmdbIdByImdbId, err := getTMDBIdsForIMDBIds(ud.TMDBTokenId, imdbIdsToFindTmdbIds)
		if err != nil {
			log.Error("failed to fetch tmdb ids for imdb ids", "error", err, "count", len(imdbIdsToFindTmdbIds))
		} else {
			for i := range items {
				item := &items[i]
				if tmdbId, ok := tmdbIdByImdbId[item.Id]; ok && tmdbId != "" {
					item.Id = "tmdb:" + tmdbId
				}
			}
		}
	}

	shouldShuffle := ud.Shuffle
	if !shouldShuffle && len(ud.ListShuffle) > 0 {
		if idx := slices.Index(ud.Lists, service+":"+id); idx != -1 {
			shouldShuffle = ud.ListShuffle[idx] == 1
		}
	}

	if shouldShuffle {
		rand.Shuffle(len(items), func(i, j int) {
			items[i], items[j] = items[j], items[i]
		})
	}

	res := stremio.CatalogHandlerResponse{
		Metas: items,
	}
	SendResponse(w, r, 200, res)
}
