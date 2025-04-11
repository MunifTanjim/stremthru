package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/shared"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
)

type DebugTorrentsData struct {
	Items      []ti.DebugTorrentsItem `json:"items"`
	TotalItems int                    `json:"total_items"`
}

func handleDebugTorrents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	q := r.URL.Query()
	noApproxSize := q.Get("no_approx_size") != ""
	noMissingSize := q.Get("no_missing_size") != ""

	items, err := ti.DebugTorrents(noApproxSize, noMissingSize)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendResponse(w, r, 200, DebugTorrentsData{
		Items:      items,
		TotalItems: len(items),
	}, nil)
}

func AddDebugEndpoints(mux *http.ServeMux) {
	withAdminAuth := shared.Middleware(AdminAuthed)

	mux.HandleFunc("/__debug__/torrents", withAdminAuth(handleDebugTorrents))
}
