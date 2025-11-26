package dash_api

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_review"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type TorrentItemFile struct {
	Path string `json:"path"`
	Idx  int    `json:"index"`
	Size string `json:"size"`
	Name string `json:"name"`
	SId  string `json:"sid,omitempty"`
	ASId string `json:"asid,omitempty"`
}

type TorrentItem struct {
	Hash    string            `json:"hash"`
	Name    string            `json:"name"`
	Size    string            `json:"size"`
	Seeders int               `json:"seeders"`
	Files   []TorrentItemFile `json:"files,omitempty"`
	Private bool              `json:"private"`
}

func handleGetTorrentsByIMDBId(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	imdbId := r.URL.Query().Get("imdbid")
	if imdbId == "" {
		ErrorBadRequest(r, "missing imdbid query parameter").Send(w, r)
		return
	}

	data, err := torrent_info.ListByStremId(imdbId, false)
	if err != nil {
		SendError(w, r, err)
		return
	}

	items := make([]TorrentItem, len(data.Items))
	for i, item := range data.Items {
		t := TorrentItem{
			Hash:    item.Hash,
			Name:    item.TorrentTitle,
			Size:    util.ToSize(item.Size),
			Seeders: item.Seeders,
			Private: item.Private,
		}
		if len(item.Files) > 0 {
			t.Files = make([]TorrentItemFile, len(item.Files))
			for i := range item.Files {
				f := item.Files[i]
				tf := TorrentItemFile{
					Path: f.Path,
					Idx:  f.Idx,
					Size: util.ToSize(f.Size),
					Name: f.Name,
					SId:  f.SId,
					ASId: f.ASId,
				}
				t.Files[i] = tf
			}
		}
		items[i] = t
	}

	SendData(w, r, 200, items)
}

var rootPeer = peer.NewAPIClient(&peer.APIClientConfig{
	BaseURL: "https://" + config.RootHost,
})

func handleSubmitTorrentReview(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	payload := struct {
		Items []torrent_review.InsertItem `json:"items"`
	}{}

	if err := ReadRequestBodyJSON(r, &payload); err != nil {
		SendError(w, r, err)
		return
	}

	machineIp := config.IP.GetMachineIP()
	for i := range payload.Items {
		item := &payload.Items[i]
		item.IP = machineIp
	}

	params := &peer.RequestTorrentReviewParams{
		Items: payload.Items,
	}

	if _, err := rootPeer.RequestTorrentReview(params); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func AddTorrentsEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/torrents", authed(handleGetTorrentsByIMDBId))
	router.HandleFunc("/torrents/review", authed(handleSubmitTorrentReview))
}
