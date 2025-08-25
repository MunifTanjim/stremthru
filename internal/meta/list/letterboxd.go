package meta_list

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/letterboxd"
	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

func handleGetLetterboxdListBySlug(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}
	userSlug := r.PathValue("user_slug")
	listSlug := r.PathValue("list_slug")

	l := letterboxd.LetterboxdList{UserName: userSlug, Slug: listSlug}

	if err := l.Fetch(); err != nil {
		SendError(w, r, err)
		return
	}

	list := List{
		Provider:    ProviderLetterboxd,
		Id:          l.Id,
		Slug:        l.Slug,
		UserId:      l.UserId,
		UserSlug:    l.UserName,
		Title:       l.Name,
		Description: l.Description,
		ItemType:    ItemTypeMovie,
		IsPrivate:   l.Private,
		IsPersonal:  false,
		UpdatedAt:   l.UpdatedAt.Time,
		Items:       []ListItem{},
	}

	letterboxdIds := make([]string, len(l.Items))

	for i := range l.Items {
		item := &l.Items[i]
		letterboxdIds[i] = item.Id
		list.Items = append(list.Items, ListItem{
			Type:        ItemTypeMovie,
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
		SendError(w, r, err)
		return
	}

	for i := range list.Items {
		item := &list.Items[i]
		if idMap, ok := idMapById[item.Id]; ok {
			item.IdMap = meta.IdMap{
				Type:       meta.IdType(idMap.Type.ToSimple()),
				IMDB:       idMap.IMDBId,
				TMDB:       idMap.TMDBId,
				TVDB:       idMap.TVDBId,
				Trakt:      idMap.TraktId,
				Letterboxd: idMap.LetterboxdId,
			}
			if idMap.MALId != "" {
				item.IdMap.Anime = &meta.IdMapAnime{
					MAL: idMap.MALId,
				}
			}
		}
	}

	SendResponse(w, r, 200, list)
}
