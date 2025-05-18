package stremio_list

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

type ExtraData struct {
	Skip  int
	Genre string
}

func getExtra(r *http.Request) *ExtraData {
	extra := &ExtraData{}
	if extraParams := GetPathParam(r, "extra", true); extraParams != "" {
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

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	catalogId := GetPathParam(r, "id", true)

	idr := parseId(catalogId)

	items := []stremio.MetaPreview{}

	switch idr.Service {
	case "mdblist":
		id, err := strconv.Atoi(idr.Id)
		if err != nil {
			shared.ErrorBadRequest(r, "invalid id").Send(w, r)
			return
		}
		list, err := ud.FetchListById(id)
		if err != nil {
			SendError(w, r, err)
			return
		}
		for i := range list.Items {
			item := &list.Items[i]
			meta := stremio.MetaPreview{
				Id:          item.ImdbId,
				Type:        mediaTypeToResourceType(item.Mediatype),
				Name:        item.Title,
				Poster:      item.Poster,
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
	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	res := stremio.CatalogHandlerResponse{
		Metas: items,
	}
	SendResponse(w, r, 200, res)
}
