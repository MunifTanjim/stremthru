package endpoint

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torznab"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

func sendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any, o string) {
	switch o {
	case "json":
		shared.SendJSON(w, r, statusCode, data)
	case "xml", "":
		shared.SendXML(w, r, statusCode, data)
	default:
		shared.SendXML(w, r, 200, znab.ErrorIncorrectParameter("invalid output format"))
	}
}

func handleTorznab(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")

	if t == "" {
		http.Redirect(w, r, r.URL.Path+"?t=caps", http.StatusTemporaryRedirect)
		return
	}

	o := strings.ToLower(r.URL.Query().Get("o"))
	if o != "" && o != "json" && o != "xml" {
		shared.SendXML(w, r, 200, znab.ErrorIncorrectParameter("invalid output format"))
		return
	}

	switch t {
	case "caps":
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendResponse(w, r, 200, torznab.StremThruIndexer.Capabilities(), o)
	case "search", "tvsearch", "movie":
		query, err := torznab.ParseQuery(r.URL.Query())
		if err != nil {
			sendResponse(w, r, 200, znab.ErrorIncorrectParameter(err.Error()), o)
			return
		}
		items, err := torznab.StremThruIndexer.Search(query)
		if err != nil {
			sendResponse(w, r, 200, znab.ErrorUnknownError(err.Error()), o)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendResponse(w, r, 200, torznab.Feed{
			Info:  torznab.StremThruIndexer.Info(),
			Items: items,
		}, o)
	default:
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendResponse(w, r, 200, znab.ErrorIncorrectParameter(t), o)
	}
}
func AddTorznabEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasTorrentInfo() {
		return
	}

	mux.HandleFunc("/v0/torznab/api", handleTorznab)
}
