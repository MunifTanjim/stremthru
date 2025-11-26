package dash_api

import (
	"net/http"
	"regexp"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

type IMDBAutocompleteItem struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Year  int    `json:"year"`
}

var imdbIdPattern = regexp.MustCompile(`^tt\d+$`)

func handleGetIMDBAutocomplete(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		SendData(w, r, 200, []IMDBAutocompleteItem{})
		return
	}

	var ids []string

	if imdbIdPattern.MatchString(query) {
		ids = []string{query}
	} else {
		var err error
		ids, err = imdb_title.SearchIds(query, "", 0, false, 10)
		if err != nil {
			SendError(w, r, err)
			return
		}
	}

	if len(ids) == 0 {
		SendData(w, r, 200, []IMDBAutocompleteItem{})
		return
	}

	titles, err := imdb_title.ListByIds(ids)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idxById := make(map[string]int, len(ids))
	for i, id := range ids {
		idxById[id] = i
	}

	items := make([]IMDBAutocompleteItem, len(titles))
	for _, title := range titles {
		idx := idxById[title.TId]
		items[idx] = IMDBAutocompleteItem{
			Id:    title.TId,
			Title: title.Title,
			Type:  title.Type,
			Year:  title.Year,
		}
	}

	SendData(w, r, 200, items)
}

func AddIMDBEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/imdb/autocomplete", authed(handleGetIMDBAutocomplete))
}
