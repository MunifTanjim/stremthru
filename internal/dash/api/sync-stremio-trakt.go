package dash_api

import (
	"net/http"
	"strings"
	"time"

	sync_stremio_trakt "github.com/MunifTanjim/stremthru/internal/sync/stremio_trakt"
)

type StremioTraktLinkResponse struct {
	StremioAccountId string                        `json:"stremio_account_id"`
	TraktAccountId   string                        `json:"trakt_account_id"`
	SyncConfig       sync_stremio_trakt.SyncConfig `json:"sync_config"`
	SyncState        sync_stremio_trakt.SyncState  `json:"sync_state"`
	CreatedAt        string                        `json:"created_at"`
	UpdatedAt        string                        `json:"updated_at"`
}

func toStremioTraktLinkResponse(item *sync_stremio_trakt.SyncStremioTraktLink) StremioTraktLinkResponse {
	resp := StremioTraktLinkResponse{
		StremioAccountId: item.StremioAccountId,
		TraktAccountId:   item.TraktAccountId,
		SyncConfig:       item.SyncConfig,
		SyncState:        item.SyncState,
		CreatedAt:        item.CAt.Format(time.RFC3339),
		UpdatedAt:        item.UAt.Format(time.RFC3339),
	}
	return resp
}

func handleGetStremioTraktLinks(w http.ResponseWriter, r *http.Request) {
	items, err := sync_stremio_trakt.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]StremioTraktLinkResponse, len(items))
	for i, item := range items {
		data[i] = toStremioTraktLinkResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateStremioTraktLinkRequest struct {
	StremioAccountId string                        `json:"stremio_account_id"`
	TraktAccountId   string                        `json:"trakt_account_id"`
	SyncConfig       sync_stremio_trakt.SyncConfig `json:"sync_config"`
}

func handleCreateStremioTraktLink(w http.ResponseWriter, r *http.Request) {
	request := &CreateStremioTraktLinkRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	errs := []Error{}
	if request.StremioAccountId == "" {
		errs = append(errs, Error{
			Location: "stremio_account_id",
			Message:  "missing stremio_account_id",
		})
	}
	if request.TraktAccountId == "" {
		errs = append(errs, Error{
			Location: "trakt_account_id",
			Message:  "missing trakt_account_id",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r, "").Append(errs...).Send(w, r)
		return
	}

	existing, err := sync_stremio_trakt.GetById(request.StremioAccountId, request.TraktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing != nil {
		ErrorBadRequest(r, "link already exists").Send(w, r)
		return
	}

	if !request.SyncConfig.Watched.Direction.IsValid() {
		ErrorBadRequest(r, "invalid sync direction").Send(w, r)
		return
	}

	link, err := sync_stremio_trakt.Link(request.StremioAccountId, request.TraktAccountId, request.SyncConfig)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toStremioTraktLinkResponse(link))
}

func parseAccountIdPair(accountIdPair string) (stremioAccountId, traktAccountId string) {
	stremioAccountId, traktAccountId, _ = strings.Cut(accountIdPair, ":")
	return stremioAccountId, traktAccountId
}

func handleGetStremioTraktLink(w http.ResponseWriter, r *http.Request) {
	stremioAccountId, traktAccountId := parseAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_trakt.GetById(stremioAccountId, traktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r, "").Send(w, r)
		return
	}

	SendData(w, r, 200, toStremioTraktLinkResponse(link))
}

type UpdateStremioTraktAccountRequest struct {
	SyncConfig sync_stremio_trakt.SyncConfig `json:"sync_config"`
}

func handleUpdateStremioTraktLink(w http.ResponseWriter, r *http.Request) {
	stremioAccountId, traktAccountId := parseAccountIdPair(r.PathValue("account_id_pair"))

	request := &UpdateStremioTraktAccountRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	link, err := sync_stremio_trakt.GetById(stremioAccountId, traktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r, "").Send(w, r)
		return
	}

	if !request.SyncConfig.Watched.Direction.IsValid() {
		ErrorBadRequest(r, "invalid sync direction").Send(w, r)
		return
	}

	if err := sync_stremio_trakt.SetSyncConfig(stremioAccountId, traktAccountId, request.SyncConfig); err != nil {
		SendError(w, r, err)
		return
	}

	link.SyncConfig = request.SyncConfig
	SendData(w, r, 200, toStremioTraktLinkResponse(link))
}

func handleDeleteStremioTraktLink(w http.ResponseWriter, r *http.Request) {
	stremioAccountId, traktAccountId := parseAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_trakt.GetById(stremioAccountId, traktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r, "").Send(w, r)
		return
	}

	if err := sync_stremio_trakt.Unlink(stremioAccountId, traktAccountId); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func handleSyncStremioTraktLink(w http.ResponseWriter, r *http.Request) {
	stremioAccountId, traktAccountId := parseAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_trakt.GetById(stremioAccountId, traktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r, "").Send(w, r)
		return
	}

	// TODO: trigger sync immediately
	SendData(w, r, 202, map[string]string{})
}

func handleResetStremioTraktLinkSyncState(w http.ResponseWriter, r *http.Request) {
	stremioAccountId, traktAccountId := parseAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_trakt.GetById(stremioAccountId, traktAccountId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r, "").Send(w, r)
		return
	}

	link.SyncState.Watched.LastSyncedAt = nil

	if err := sync_stremio_trakt.SetSyncState(
		link.StremioAccountId,
		link.TraktAccountId,
		link.SyncState,
	); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toStremioTraktLinkResponse(link))
}

func AddSyncStremioTraktEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/sync/stremio-trakt/links", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioTraktLinks(w, r)
		case http.MethodPost:
			handleCreateStremioTraktLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-trakt/links/{account_id_pair}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioTraktLink(w, r)
		case http.MethodPatch:
			handleUpdateStremioTraktLink(w, r)
		case http.MethodDelete:
			handleDeleteStremioTraktLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-trakt/links/{account_id_pair}/sync", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleSyncStremioTraktLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-trakt/links/{account_id_pair}/reset-sync-state", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleResetStremioTraktLinkSyncState(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
