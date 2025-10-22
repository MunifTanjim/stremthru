package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torznab"
)

func handleTorznab(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")

	if t == "" {
		http.Redirect(w, r, r.URL.Path+"?t=caps", http.StatusTemporaryRedirect)
		return
	}

	switch t {
	case "caps":
		w.Header().Set("Cache-Control", "public, max-age=7200")
		shared.SendXML(w, r, 200, torznab.StremThruIndexer.Capabilities())
	case "search", "tvsearch", "movie":
		query, err := torznab.ParseQuery(r.URL.Query())
		if err != nil {
			shared.SendXML(w, r, 200, torznab.ErrorIncorrectParameter(err.Error()))
			return
		}
		items, err := torznab.StremThruIndexer.Search(query)
		if err != nil {
			shared.SendXML(w, r, 200, torznab.ErrorUnknownError(err.Error()))
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=7200")
		shared.SendXML(w, r, 200, torznab.ResultFeed{
			Info:  torznab.StremThruIndexer.Info(),
			Items: items,
		})
	default:
		w.Header().Set("Cache-Control", "public, max-age=7200")
		shared.SendXML(w, r, 200, torznab.ErrorIncorrectParameter(t))
	}
}
func AddTorznabEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasTorrentInfo() {
		return
	}

	mux.HandleFunc("/v0/torznab/api", handleTorznab)
}
