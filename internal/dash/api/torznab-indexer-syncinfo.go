package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type TorznabIndexerSyncInfoResponse struct {
	Type        string  `json:"type"`
	Id          string  `json:"id"`
	SId         string  `json:"sid"`
	QueuedAt    *string `json:"queued_at"`
	SyncedAt    *string `json:"synced_at"`
	Error       *string `json:"error"`
	ResultCount *int64  `json:"result_count"`
}

type ListTorznabIndexerSyncInfoResponse struct {
	Items      []TorznabIndexerSyncInfoResponse `json:"items"`
	TotalCount int                              `json:"total_count"`
}

func toTorznabIndexerSyncInfoResponse(item *torznab_indexer_syncinfo.TorznabIndexerSyncInfo) TorznabIndexerSyncInfoResponse {
	res := TorznabIndexerSyncInfoResponse{
		Type: string(item.Type),
		Id:   item.Id,
		SId:  item.SId,
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

func AddTorznabIndexerSyncInfoEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/torrents/indexer-syncinfos", authed(handleGetTorznabIndexerSyncInfos))
}
