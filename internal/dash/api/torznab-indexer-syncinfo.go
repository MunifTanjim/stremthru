package dash_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

type TorznabIndexerSyncInfoQueryResponse struct {
	Query string `json:"query"`
	Done  bool   `json:"done"`
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

type TorznabIndexerSyncInfoResponse struct {
	Type        string                                `json:"type"`
	Id          string                                `json:"id"`
	SId         string                                `json:"sid"`
	QueuedAt    *string                               `json:"queued_at"`
	SyncedAt    *string                               `json:"synced_at"`
	Error       *string                               `json:"error"`
	ResultCount *int64                                `json:"result_count"`
	Status      string                                `json:"status"`
	Queries     []TorznabIndexerSyncInfoQueryResponse `json:"queries"`
}

type ListTorznabIndexerSyncInfoResponse struct {
	Items      []TorznabIndexerSyncInfoResponse `json:"items"`
	TotalCount int                              `json:"total_count"`
}

func toTorznabIndexerSyncInfoResponse(item *torznab_indexer_syncinfo.TorznabIndexerSyncInfo) TorznabIndexerSyncInfoResponse {
	res := TorznabIndexerSyncInfoResponse{
		Type:    string(item.Type),
		Id:      item.Id,
		SId:     item.SId,
		Status:  string(item.Status),
		Queries: make([]TorznabIndexerSyncInfoQueryResponse, len(item.Queries)),
	}

	for i, q := range item.Queries {
		res.Queries[i] = TorznabIndexerSyncInfoQueryResponse{
			Query: q.Query,
			Done:  q.Done,
			Count: q.Count,
			Error: q.Error,
		}
	}

	if !item.QueuedAt.Time.IsZero() {
		queuedAt := item.QueuedAt.Time.Format(time.RFC3339)
		res.QueuedAt = &queuedAt
	}

	if !item.SyncedAt.Time.IsZero() {
		syncedAt := item.SyncedAt.Time.Format(time.RFC3339)
		res.SyncedAt = &syncedAt
	}

	if !item.Error.IsZero() {
		res.Error = &item.Error.String
	}

	if item.ResultCount.Valid {
		res.ResultCount = &item.ResultCount.Int64
	}

	return res
}

func handleGetTorznabIndexerSyncInfos(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query()

	limit := util.SafeParseInt(query.Get("limit"), 0)
	offset := util.SafeParseInt(query.Get("offset"), 0)
	sid := query.Get("sid")

	items, err := torznab_indexer_syncinfo.GetItems(torznab_indexer_syncinfo.GetItemsParams{
		Limit:  limit,
		Offset: offset,
		SId:    sid,
	})
	if err != nil {
		SendError(w, r, err)
		return
	}

	totalCount, err := torznab_indexer_syncinfo.CountItems(sid)
	if err != nil {
		SendError(w, r, err)
		return
	}

	responseItems := make([]TorznabIndexerSyncInfoResponse, len(items))
	for i := range items {
		responseItems[i] = toTorznabIndexerSyncInfoResponse(&items[i])
	}

	data := ListTorznabIndexerSyncInfoResponse{
		Items:      responseItems,
		TotalCount: totalCount,
	}

	SendData(w, r, 200, data)
}

type QueueTorznabIndexerSyncInfoRequest struct {
	SId string `json:"sid"`
}

func handleQueueTorznabIndexerSyncInfo(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	request := &QueueTorznabIndexerSyncInfoRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	nsid, err := torrent_stream.NormalizeStreamId(request.SId)
	if !strings.HasPrefix(request.SId, "tt") || err != nil {
		ErrorBadRequest(r, "Invalid IMDB Id").Send(w, r)
		return
	}

	if title, err := imdb_title.Get(nsid.Id); err != nil {
		SendError(w, r, err)
		return
	} else if title == nil {
		ErrorBadRequest(r, "Unknow IMDB Id").Send(w, r)
		return
	}

	// Queue the sync request - the queue worker will prepare queries and create syncinfo entries
	worker_queue.TorznabIndexerSyncerQueue.Queue(worker_queue.TorznabIndexerSyncerQueueItem{
		SId: request.SId,
	})

	SendData(w, r, 204, nil)
}

func AddTorznabIndexerSyncInfoEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/torrents/indexer-syncinfos", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTorznabIndexerSyncInfos(w, r)
		case http.MethodPost:
			handleQueueTorznabIndexerSyncInfo(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
