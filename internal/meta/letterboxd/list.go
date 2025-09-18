package meta_letterboxd

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/letterboxd"
	meta_type "github.com/MunifTanjim/stremthru/internal/meta/type"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

func toList(l *letterboxd.LetterboxdList) (*meta_type.List, error) {
	list := meta_type.List{
		Provider:    meta_type.ProviderLetterboxd,
		Id:          l.Id,
		Slug:        l.Slug,
		UserId:      l.UserId,
		UserSlug:    l.UserName,
		Title:       l.Name,
		Description: l.Description,
		ItemType:    meta_type.ItemTypeMovie,
		IsPrivate:   l.Private,
		IsPersonal:  false,
		ItemCount:   l.ItemCount,
		UpdatedAt:   l.UpdatedAt.Time,
		Items:       []meta_type.ListItem{},
	}

	letterboxdIds := make([]string, len(l.Items))

	for i := range l.Items {
		item := &l.Items[i]
		letterboxdIds[i] = item.Id
		list.Items = append(list.Items, meta_type.ListItem{
			Type:        meta_type.ItemTypeMovie,
			Id:          item.Id,
			Slug:        "",
			Title:       item.Name,
			Description: "",
			Year:        item.ReleaseYear,
			IsAdult:     item.Adult,
			Runtime:     item.Runtime,
			Rating:      item.Rating,
			Poster:      item.Poster,
			UpdatedAt:   item.UpdatedAt.Time,
			Index:       i,
			GenreIds:    item.GenreIds,
		})
	}

	idMapById, err := imdb_title.GetIdMapsByLetterboxdId(letterboxdIds)
	if err != nil {
		return nil, err
	}

	for i := range list.Items {
		item := &list.Items[i]
		if idMap, ok := idMapById[item.Id]; ok {
			item.IdMap = meta_type.IdMap{
				Type:       meta_type.IdType(idMap.Type.ToSimple()),
				IMDB:       idMap.IMDBId,
				TMDB:       idMap.TMDBId,
				TVDB:       idMap.TVDBId,
				Trakt:      idMap.TraktId,
				Letterboxd: idMap.LetterboxdId,
			}
			if idMap.MALId != "" {
				item.IdMap.Anime = &meta_type.IdMapAnime{
					MAL: idMap.MALId,
				}
			}
		}
	}

	return &list, nil
}

func handleGetLetterboxdListById(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	l := letterboxd.LetterboxdList{Id: r.PathValue("list_id")}
	if err := l.Fetch(); err != nil {
		SendError(w, r, err)
		return
	}

	list, err := toList(&l)
	if err != nil {
		SendError(w, r, err)
		return
	}

	cacheMaxAge := min(int64(l.StaleIn().Seconds()), int64(2*time.Hour.Seconds()))
	w.Header().Add("Cache-Control", "public, max-age="+strconv.FormatInt(cacheMaxAge, 10))

	SendResponse(w, r, 200, list, nil)
}

func handleGetLetterboxdUserWatchlist(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	userId := r.PathValue("user_id")
	l := letterboxd.LetterboxdList{Id: letterboxd.ID_PREFIX_USER_WATCHLIST + userId}
	if err := l.Fetch(); err != nil {
		SendError(w, r, err)
		return
	}

	list, err := toList(&l)
	if err != nil {
		SendError(w, r, err)
		return
	}

	cacheMaxAge := min(int64(l.StaleIn().Seconds()), int64(2*time.Hour.Seconds()))
	w.Header().Add("Cache-Control", "public, max-age="+strconv.FormatInt(cacheMaxAge, 10))

	SendResponse(w, r, 200, list, nil)
}
