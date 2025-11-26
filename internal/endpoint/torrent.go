package endpoint

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer_token"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_review"
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

var torrentStatsCached = cache.NewCachedValue(cache.CachedValueConfig[*torrent_info.Stats]{
	Get: torrent_info.GetStats,
	TTL: 6 * time.Hour,
})

func handleTorrentStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}
	stats, err := torrentStatsCached.Get()
	if err != nil {
		SendError(w, r, err)
		return
	}
	cacheMaxAge := strconv.Itoa(int(time.Until(torrentStatsCached.StaleAt()).Seconds()))
	w.Header().Add("Cache-Control", "max-age="+cacheMaxAge+"")
	SendResponse(w, r, 200, stats, nil)
}

var torrentReviewRequestMutex sync.Mutex

var torrentReviewRequestRateLimit = cache.NewCache[int](&cache.CacheConfig{
	Lifetime:      10 * time.Minute,
	Name:          "torrent_review_request_rate_limit",
	LocalCapacity: 2048,
})

var torrentReviewRequestGlobalRateLimit = cache.NewCache[int](&cache.CacheConfig{
	Lifetime:      2 * time.Minute,
	Name:          "torrent_review_request_global_rate_limit",
	LocalCapacity: 1,
})

type RequestTorrentReviewPayload struct {
	Items []torrent_review.InsertItem
}

func handleRequestTorrentReview(w http.ResponseWriter, r *http.Request) {
	torrentReviewRequestMutex.Lock()
	defer torrentReviewRequestMutex.Unlock()

	clientIP := core.GetClientIP(r)

	var globalRequestCount int
	torrentReviewRequestGlobalRateLimit.Get("", &globalRequestCount)
	if globalRequestCount >= 2 {
		shared.ErrorTooManyRequests(r, "global rate limit exceeded").Send(w, r)
		return
	}
	defer torrentReviewRequestGlobalRateLimit.Add("", globalRequestCount+1)

	rateLimitKey := clientIP
	var requestCount int
	torrentReviewRequestRateLimit.Get(rateLimitKey, &requestCount)
	if requestCount >= 3 {
		shared.ErrorTooManyRequests(r, "rate limit exceeded").Send(w, r)
		return
	}
	defer torrentReviewRequestRateLimit.Add(rateLimitKey, requestCount+1)

	payload := RequestTorrentReviewPayload{}
	if err := shared.ReadRequestBodyJSON(r, &payload); err != nil {
		SendError(w, r, err)
		return
	}

	for i := range payload.Items {
		item := &payload.Items[i]
		item.IP = clientIP
	}

	if err := torrent_review.Insert(payload.Items); err != nil {
		SendError(w, r, err)
		return
	}

	SendResponse(w, r, 204, nil, nil)
}

func handleTorrentReviews(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleRequestTorrentReview(w, r)
	default:
		shared.ErrorMethodNotAllowed(r).Send(w, r)
	}
}

func AddTorrentEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasTorrentInfo() {
		return
	}

	mux.HandleFunc("/v0/torrents", handleTorrents)
	mux.HandleFunc("/v0/torrents/stats", handleTorrentStats)
	mux.HandleFunc("/v0/torrents/review", handleTorrentReviews)
}
