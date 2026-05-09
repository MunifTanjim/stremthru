package dash_api

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type AniDBAutocompleteItem struct {
	Id     string `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Season string `json:"season"`
	Year   string `json:"year"`
}

func handleGetAniDBAutocomplete(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		SendData(w, r, 200, []AniDBAutocompleteItem{})
		return
	}

	var ids []string

	if util.IsNumericString(query) {
		ids = []string{query}
	} else {
		var err error
		ids, err = anidb.SearchIdsByTitle(query, nil, 0, 10)
		if err != nil {
			SendError(w, r, err)
			return
		}
	}

	if len(ids) == 0 {
		SendData(w, r, 200, []AniDBAutocompleteItem{})
		return
	}

	titles, err := anidb.GetTitlesByIds(ids)
	if err != nil {
		SendError(w, r, err)
		return
	}

	// Dedupe by TId, prefer title that best matches the query
	normalizer := util.NewStringNormalizer()
	itemById := make(map[string]AniDBAutocompleteItem)
	scoreById := make(map[string]int)

	for _, t := range titles {
		score := util.FuzzyTokenSetRatio(query, t.Value, normalizer)

		if score > scoreById[t.TId] {
			scoreById[t.TId] = score
			itemById[t.TId] = AniDBAutocompleteItem{
				Id:     t.TId,
				Title:  t.Value,
				Type:   t.Type,
				Season: t.Season,
				Year:   t.Year,
			}
		}
	}

	// Preserve order from search results
	items := make([]AniDBAutocompleteItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := itemById[id]; ok {
			items = append(items, item)
		}
	}

	SendData(w, r, 200, items)
}

func AddAniDBEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/anidb/autocomplete", authed(handleGetAniDBAutocomplete))
}
