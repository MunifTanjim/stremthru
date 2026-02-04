package dash_api

import (
	"net/http"
	"strings"
	"time"

	sync_stremio_stremio "github.com/MunifTanjim/stremthru/internal/sync/stremio_stremio"
)

type StremioStremioLinkResponse struct {
	AccountAId string                          `json:"account_a_id"`
	AccountBId string                          `json:"account_b_id"`
	SyncConfig sync_stremio_stremio.SyncConfig `json:"sync_config"`
	SyncState  sync_stremio_stremio.SyncState  `json:"sync_state"`
	CreatedAt  string                          `json:"created_at"`
	UpdatedAt  string                          `json:"updated_at"`
}

func toStremioStremioLinkResponse(item *sync_stremio_stremio.SyncStremioStremioLink) StremioStremioLinkResponse {
	resp := StremioStremioLinkResponse{
		AccountAId: item.AccountAId,
		AccountBId: item.AccountBId,
		SyncConfig: item.SyncConfig,
		SyncState:  item.SyncState,
		CreatedAt:  item.CAt.Format(time.RFC3339),
		UpdatedAt:  item.UAt.Format(time.RFC3339),
	}
	return resp
}

func handleGetStremioStremioLinks(w http.ResponseWriter, r *http.Request) {
	items, err := sync_stremio_stremio.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]StremioStremioLinkResponse, len(items))
	for i, item := range items {
		data[i] = toStremioStremioLinkResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateStremioStremioLinkRequest struct {
	AccountAId string                          `json:"account_a_id"`
	AccountBId string                          `json:"account_b_id"`
	SyncConfig sync_stremio_stremio.SyncConfig `json:"sync_config"`
}

func handleCreateStremioStremioLink(w http.ResponseWriter, r *http.Request) {
	request := &CreateStremioStremioLinkRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	errs := []Error{}
	if request.AccountAId == "" {
		errs = append(errs, Error{
			Location: "account_a_id",
			Message:  "missing account_a_id",
		})
	}
	if request.AccountBId == "" {
		errs = append(errs, Error{
			Location: "account_b_id",
			Message:  "missing account_b_id",
		})
	}
	if request.AccountAId == request.AccountBId {
		errs = append(errs, Error{
			Location: "account_b_id",
			Message:  "account_a_id and account_b_id must be different",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r).Append(errs...).Send(w, r)
		return
	}

	existing, err := sync_stremio_stremio.GetById(request.AccountAId, request.AccountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing != nil {
		ErrorBadRequest(r).WithMessage("link already exists").Send(w, r)
		return
	}

	if !request.SyncConfig.Watched.Direction.IsValid() {
		ErrorBadRequest(r).WithMessage("invalid sync direction").Send(w, r)
		return
	}

	link, err := sync_stremio_stremio.Link(request.AccountAId, request.AccountBId, request.SyncConfig)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toStremioStremioLinkResponse(link))
}

func parseStremioAccountIdPair(accountIdPair string) (accountAId, accountBId string) {
	accountAId, accountBId, _ = strings.Cut(accountIdPair, ":")
	return accountAId, accountBId
}

func handleGetStremioStremioLink(w http.ResponseWriter, r *http.Request) {
	accountAId, accountBId := parseStremioAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_stremio.GetById(accountAId, accountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	SendData(w, r, 200, toStremioStremioLinkResponse(link))
}

type UpdateStremioStremioLinkRequest struct {
	SyncConfig sync_stremio_stremio.SyncConfig `json:"sync_config"`
}

func handleUpdateStremioStremioLink(w http.ResponseWriter, r *http.Request) {
	accountAId, accountBId := parseStremioAccountIdPair(r.PathValue("account_id_pair"))

	request := &UpdateStremioStremioLinkRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	link, err := sync_stremio_stremio.GetById(accountAId, accountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if !request.SyncConfig.Watched.Direction.IsValid() {
		ErrorBadRequest(r).WithMessage("invalid sync direction").Send(w, r)
		return
	}

	if err := sync_stremio_stremio.SetSyncConfig(accountAId, accountBId, request.SyncConfig); err != nil {
		SendError(w, r, err)
		return
	}

	link.SyncConfig = request.SyncConfig
	SendData(w, r, 200, toStremioStremioLinkResponse(link))
}

func handleDeleteStremioStremioLink(w http.ResponseWriter, r *http.Request) {
	accountAId, accountBId := parseStremioAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_stremio.GetById(accountAId, accountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if err := sync_stremio_stremio.Unlink(accountAId, accountBId); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func handleSyncStremioStremioLink(w http.ResponseWriter, r *http.Request) {
	accountAId, accountBId := parseStremioAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_stremio.GetById(accountAId, accountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	// TODO: trigger sync immediately
	SendData(w, r, 202, map[string]string{})
}

func handleResetStremioStremioLinkSyncState(w http.ResponseWriter, r *http.Request) {
	accountAId, accountBId := parseStremioAccountIdPair(r.PathValue("account_id_pair"))

	link, err := sync_stremio_stremio.GetById(accountAId, accountBId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if link == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	link.SyncState.Watched.LastSyncedAt = nil

	if err := sync_stremio_stremio.SetSyncState(
		link.AccountAId,
		link.AccountBId,
		link.SyncState,
	); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toStremioStremioLinkResponse(link))
}

func AddSyncStremioStremioEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/sync/stremio-stremio/links", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioStremioLinks(w, r)
		case http.MethodPost:
			handleCreateStremioStremioLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-stremio/links/{account_id_pair}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioStremioLink(w, r)
		case http.MethodPatch:
			handleUpdateStremioStremioLink(w, r)
		case http.MethodDelete:
			handleDeleteStremioStremioLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-stremio/links/{account_id_pair}/sync", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleSyncStremioStremioLink(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/sync/stremio-stremio/links/{account_id_pair}/reset-sync-state", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleResetStremioStremioLinkSyncState(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
