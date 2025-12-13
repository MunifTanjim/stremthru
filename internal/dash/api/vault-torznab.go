package dash_api

import (
	"errors"
	"net/http"
	"time"

	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
)

type TorznabIndexerResponse struct {
	Type      string `json:"type"`
	Id        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	IsValid   bool   `json:"is_valid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toTorznabIndexerResponse(item *torznab_indexer.TorznabIndexer) TorznabIndexerResponse {
	compositeId := string(item.Type) + ":" + item.Id
	return TorznabIndexerResponse{
		Type:      string(item.Type),
		Id:        compositeId,
		Name:      item.Name,
		URL:       item.URL,
		CreatedAt: item.CAt.Format(time.RFC3339),
		UpdatedAt: item.UAt.Format(time.RFC3339),
	}
}

func handleGetTorznabIndexers(w http.ResponseWriter, r *http.Request) {
	items, err := torznab_indexer.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]TorznabIndexerResponse, len(items))
	for i := range items {
		data[i] = toTorznabIndexerResponse(&items[i])
	}

	SendData(w, r, 200, data)
}

type CreateTorznabIndexerRequest struct {
	Type   torznab_indexer.IndexerType `json:"type"`
	URL    string                      `json:"url"`
	APIKey string                      `json:"api_key"`
	Name   string                      `json:"name,omitempty"`
}

var ErrorInvalidTorznabCredentials = errors.New("invalid torznab credentials or connection failed")

func handleCreateTorznabIndexer(w http.ResponseWriter, r *http.Request) {
	request := &CreateTorznabIndexerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	errs := []Error{}
	if request.URL == "" {
		errs = append(errs, Error{
			Location: "url",
			Message:  "missing url",
		})
	}
	if request.APIKey == "" {
		errs = append(errs, Error{
			Location: "api_key",
			Message:  "missing api_key",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r, "").Append(errs...).Send(w, r)
		return
	}

	indexerType := request.Type
	if indexerType == "" {
		indexerType = torznab_indexer.IndexerTypeJackett
	}

	indexer, err := torznab_indexer.NewTorznabIndexer(indexerType, request.URL, request.APIKey)
	if err != nil {
		ErrorBadRequest(r, "Invalid Torznab URL").WithCause(err).Send(w, r)
		return
	}

	if request.Name != "" {
		indexer.Name = request.Name
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r, "Invalid Torznab URL or API key").Send(w, r)
		return
	}

	if err := indexer.Upsert(); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toTorznabIndexerResponse(indexer))
}

func handleGetTorznabIndexer(w http.ResponseWriter, r *http.Request) {
	compositeId := r.PathValue("id")

	indexer, err := torznab_indexer.GetByCompositeId(compositeId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r, "torznab indexer not found").Send(w, r)
		return
	}

	SendData(w, r, 200, toTorznabIndexerResponse(indexer))
}

type UpdateTorznabIndexerRequest struct {
	APIKey string `json:"api_key"`
	Name   string `json:"name,omitempty"`
}

func handleUpdateTorznabIndexer(w http.ResponseWriter, r *http.Request) {
	compositeId := r.PathValue("id")

	request := &UpdateTorznabIndexerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	indexer, err := torznab_indexer.GetByCompositeId(compositeId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r, "indexer not found").Send(w, r)
		return
	}

	if request.APIKey != "" {
		indexer.SetAPIKey(request.APIKey)
	}

	if request.Name != "" {
		indexer.Name = request.Name
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r, "Invalid Torznab API key").Send(w, r)
		return
	}

	if err := indexer.Upsert(); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toTorznabIndexerResponse(indexer))
}

func handleDeleteTorznabIndexer(w http.ResponseWriter, r *http.Request) {
	compositeId := r.PathValue("id")

	existing, err := torznab_indexer.GetByCompositeId(compositeId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r, "torznab indexer not found").Send(w, r)
		return
	}

	if err := torznab_indexer.DeleteByCompositeId(compositeId); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func handleTestTorznabIndexer(w http.ResponseWriter, r *http.Request) {
	compositeId := r.PathValue("id")

	indexer, err := torznab_indexer.GetByCompositeId(compositeId)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r, "torznab indexer not found").Send(w, r)
		return
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r, "Connection test failed").Send(w, r)
		return
	}

	SendData(w, r, 200, toTorznabIndexerResponse(indexer))
}

func AddVaultTorznabEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/vault/torznab/indexers", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTorznabIndexers(w, r)
		case http.MethodPost:
			handleCreateTorznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/torznab/indexers/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTorznabIndexer(w, r)
		case http.MethodPatch:
			handleUpdateTorznabIndexer(w, r)
		case http.MethodDelete:
			handleDeleteTorznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/torznab/indexers/{id}/test", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleTestTorznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
