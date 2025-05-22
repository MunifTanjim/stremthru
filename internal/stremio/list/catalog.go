package stremio_list

import (
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
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

	go imdb_title.BulkRecordMappingFromMDBList(newMappings)
	if err = imdb_title.UpsertMetas(newMetas); err != nil {
		return nil, err
	}
	return byId, nil
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

	items := []stremio.MetaPreview{}

	switch service {
	case "mdblist":
		id, err := strconv.Atoi(id)
		if err != nil {
			shared.ErrorBadRequest(r, "invalid id").Send(w, r)
			return
		}
		list := mdblist.MDBListList{Id: id}
		if err := ud.FetchMDBListList(&list); err != nil {
			SendError(w, r, err)
			return
		}

		rpdbPosterBaseUrl := ""
		if ud.RPDBAPIKey != "" {
			rpdbPosterBaseUrl = "https://api.ratingposterdb.com/" + ud.RPDBAPIKey + "/imdb/poster-default/"
		}

		for i := range list.Items {
			item := &list.Items[i]

			poster := item.Poster
			if rpdbPosterBaseUrl != "" {
				poster = rpdbPosterBaseUrl + item.IMDBId + ".jpg?fallback=true"
			}

			meta := stremio.MetaPreview{
				Id:          item.IMDBId,
				Type:        mediaTypeToResourceType(item.Mediatype),
				Name:        item.Title,
				Poster:      poster,
				PosterShape: stremio.MetaPosterShapePoster,
				Genres:      item.Genre,
				ReleaseInfo: strconv.Itoa(item.ReleaseYear),
			}
			items = append(items, meta)
		}
	default:
		shared.ErrorBadRequest(r, "invalid id").Send(w, r)
		return
	}

	extra := getExtra(r)

	if extra.Genre != "" {
		filteredItems := []stremio.MetaPreview{}
		for i := range items {
			item := &items[i]
			if slices.Contains(item.Genres, extra.Genre) {
				filteredItems = append(filteredItems, *item)
			}
		}
		items = filteredItems
	}

	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	imdbIds := []string{}
	for i := range items {
		imdbIds = append(imdbIds, items[i].Id)
	}

	metaById, err := getIMDBMetaFromMDBList(imdbIds, ud.MDBListAPIkey)
	if err != nil {
		SendError(w, r, err)
		return
	}

	for i := range items {
		item := &items[i]
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
	}

	if ud.Shuffle {
		rand.Shuffle(len(items), func(i, j int) {
			items[i], items[j] = items[j], items[i]
		})
	}

	res := stremio.CatalogHandlerResponse{
		Metas: items,
	}
	SendResponse(w, r, 200, res)
}
