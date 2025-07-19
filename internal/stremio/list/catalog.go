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
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/stremio"
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

			poster := media.Cover

			meta := stremio.MetaPreview{
				Type:        "anime",
				Name:        media.Title,
				Description: media.Description,
				Poster:      poster,
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
				Genres:      item.Genre,
				ReleaseInfo: strconv.Itoa(item.ReleaseYear),
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

			if media.IdMap == nil || media.IdMap.Kitsu == "" {
				continue
			}

			item.Id = "kitsu:" + media.IdMap.Kitsu
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
