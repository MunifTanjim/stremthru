package endpoint

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/shared"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
)

func handleExperimentZileanTorrents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	q := r.URL.Query()
	noApproxSize := q.Get("no_approx_size") != ""
	noMissingSize := q.Get("no_missing_size") != ""
	excludeSource := strings.Split(q.Get("exclude_source"), ",")

	items, err := ti.DumpTorrents(noApproxSize, noMissingSize, excludeSource)
	if err != nil {
		SendError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(items); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func AddExperimentEndpoints(mux *http.ServeMux) {
	withAdminAuth := shared.Middleware(AdminAuthed)

	mux.HandleFunc("/__experiment__/zilean/torrents", withAdminAuth(handleExperimentZileanTorrents))
}
