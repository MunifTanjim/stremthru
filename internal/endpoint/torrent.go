package endpoint

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer_token"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"golang.org/x/sync/singleflight"
)

type RecordTorrentsPayload struct {
	Items []torrent_info.TorrentItem `json:"items"`
}

func handleRecordTorrents(w http.ResponseWriter, r *http.Request) {
	peerToken := r.Header.Get("X-StremThru-Peer-Token")
	isValidToken, err := peer_token.IsValid(peerToken)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if !isValidToken {
		shared.ErrorUnauthorized(r).Send(w, r)
		return
	}

	payload := &RecordTorrentsPayload{}
	if err := shared.ReadRequestBodyJSON(r, payload); err != nil {
		SendError(w, r, err)
		return
	}

	go torrent_info.Upsert(payload.Items, "", false)
	w.WriteHeader(204)
}

func handleListTorrents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sid := query.Get("sid")
	if sid == "" {
		shared.ErrorBadRequest(r, "missing sid").Send(w, r)
		return
	}
	if !strings.HasPrefix(sid, "tt") && !strings.HasPrefix(sid, "anidb:") {
		shared.ErrorBadRequest(r, "unsupported sid").Send(w, r)
		return
	}

	originInstanceId := r.Header.Get(server.HEADER_ORIGIN_INSTANCE_ID)
	if originInstanceId == "" {
		w.Header().Set(server.HEADER_ORIGIN_INSTANCE_ID, originInstanceId)
	} else {
		w.Header().Set(server.HEADER_ORIGIN_INSTANCE_ID, config.InstanceId)
	}

	localOnly := query.Get("local_only") != ""
	noMissingSize := query.Get("no_missing_size") != ""
	data, err := buddy.ListTorrentsByStremId(sid, localOnly, originInstanceId, noMissingSize)

	w.Header().Set("Cache-Control", "public, max-age=7200")
	SendResponse(w, r, 200, data, err)
}

func handleTorrents(w http.ResponseWriter, r *http.Request) {
	if shared.IsMethod(r, http.MethodPost) {
		handleRecordTorrents(w, r)
		return
	}
	if shared.IsMethod(r, http.MethodGet) {
		handleListTorrents(w, r)
		return
	}
	shared.ErrorMethodNotAllowed(r).Send(w, r)
}

type TorrentStatsCached struct {
	stats   torrent_info.Stats
	staleAt time.Time
}

var torrentStatsCached TorrentStatsCached
var torrentStatsCachedGroup singleflight.Group

func handleTorrentStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}
	if torrentStatsCached.staleAt.Before(time.Now()) {
		_, err, _ := torrentStatsCachedGroup.Do("", func() (any, error) {
			stats, err := torrent_info.GetStats()
			if err != nil {
				return nil, err
			}
			torrentStatsCached.stats = *stats
			torrentStatsCached.staleAt = time.Now().Add(6 * time.Hour)
			return nil, nil
		})
		if err != nil {
			SendError(w, r, err)
			return
		}
	}
	cacheMaxAge := strconv.Itoa(int(torrentStatsCached.staleAt.Sub(time.Now()).Seconds()))
	w.Header().Add("Cache-Control", "max-age="+cacheMaxAge+"")
	SendResponse(w, r, 200, torrentStatsCached.stats, nil)
}

func AddTorrentEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasTorrentInfo() {
		return
	}

	mux.HandleFunc("/v0/torrents", handleTorrents)
	mux.HandleFunc("/v0/torrents/stats", handleTorrentStats)
}
