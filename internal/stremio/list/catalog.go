package stremio_list

import (
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"strconv"

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
				poster = rpdbPosterBaseUrl + item.ImdbId + ".jpg?fallback=true"
			}

			meta := stremio.MetaPreview{
				Id:          item.ImdbId,
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
