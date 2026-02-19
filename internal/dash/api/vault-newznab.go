package dash_api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	newznab_indexer "github.com/MunifTanjim/stremthru/internal/newznab/indexer"
	"github.com/MunifTanjim/stremthru/internal/ratelimit"
)

type NewznabIndexerResponse struct {
	Id                int64   `json:"id"`
	Type              string  `json:"type"`
	Name              string  `json:"name"`
	URL               string  `json:"url"`
	RateLimitConfigId *string `json:"rate_limit_config_id"`
	Disabled          bool    `json:"disabled"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toNewznabIndexerResponse(item *newznab_indexer.NewznabIndexer) NewznabIndexerResponse {
	var rateLimitConfigId *string
	if item.RateLimitConfigId.Valid {
		rateLimitConfigId = &item.RateLimitConfigId.String
	}

	return NewznabIndexerResponse{
		Id:                item.Id,
		Type:              string(item.Type),
		Name:              item.Name,
		URL:               item.URL,
		RateLimitConfigId: rateLimitConfigId,
		Disabled:          item.Disabled,
		CreatedAt:         item.CAt.Format(time.RFC3339),
		UpdatedAt:         item.UAt.Format(time.RFC3339),
	}
}

func handleGetNewznabIndexers(w http.ResponseWriter, r *http.Request) {
	items, err := newznab_indexer.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]NewznabIndexerResponse, len(items))
	for i := range items {
		data[i] = toNewznabIndexerResponse(&items[i])
	}

	SendData(w, r, 200, data)
}

type CreateNewznabIndexerRequest struct {
	URL               string  `json:"url"`
	APIKey            string  `json:"api_key"`
	Name              string  `json:"name"`
	RateLimitConfigId *string `json:"rate_limit_config_id"`
}

func handleCreateNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	request := &CreateNewznabIndexerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	errs := []Error{}
	if request.Name == "" {
		errs = append(errs, Error{
			Location: "name",
			Message:  "missing name",
		})
	}
	if request.URL == "" {
		errs = append(errs, Error{
			Location: "url",
			Message:  "missing url",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r).Append(errs...).Send(w, r)
		return
	}

	indexer, err := newznab_indexer.NewNewznabIndexer(request.URL, request.APIKey)
	if err != nil {
		ErrorBadRequest(r).WithMessage("Invalid Newznab URL").WithCause(err).Send(w, r)
		return
	}

	indexer.Name = request.Name

	if request.RateLimitConfigId != nil && *request.RateLimitConfigId != "" {
		if rlc, err := ratelimit.GetById(*request.RateLimitConfigId); err != nil {
			SendError(w, r, err)
			return
		} else if rlc == nil {
			ErrorBadRequest(r).Append(Error{
				Location: "rate_limit_config_id",
				Message:  "rate limit config not found",
			}).Send(w, r)
			return
		}
		indexer.RateLimitConfigId = sql.NullString{
			String: *request.RateLimitConfigId,
			Valid:  true,
		}
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r).WithMessage("Invalid Newznab URL or API key").WithCause(err).Send(w, r)
		return
	}

	if err := indexer.Insert(); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toNewznabIndexerResponse(indexer))
}

func handleGetNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ErrorBadRequest(r).WithMessage("invalid id").Send(w, r)
		return
	}

	indexer, err := newznab_indexer.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r).WithMessage("newznab indexer not found").Send(w, r)
		return
	}

	SendData(w, r, 200, toNewznabIndexerResponse(indexer))
}

type UpdateNewznabIndexerRequest struct {
	APIKey            string  `json:"api_key"`
	Name              string  `json:"name,omitempty"`
	RateLimitConfigId *string `json:"rate_limit_config_id"`
}

func handleUpdateNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ErrorBadRequest(r).WithMessage("invalid id").Send(w, r)
		return
	}

	request := &UpdateNewznabIndexerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	indexer, err := newznab_indexer.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r).WithMessage("newznab indexer not found").Send(w, r)
		return
	}

	if request.APIKey != "" {
		if err := indexer.SetAPIKey(request.APIKey); err != nil {
			SendError(w, r, err)
			return
		}
	}

	if request.Name != "" {
		indexer.Name = request.Name
	}

	if request.RateLimitConfigId == nil || *request.RateLimitConfigId == "" {
		indexer.RateLimitConfigId = sql.NullString{Valid: false}
	} else if config, err := ratelimit.GetById(*request.RateLimitConfigId); err != nil {
		SendError(w, r, err)
		return
	} else if config == nil {
		ErrorBadRequest(r).Append(Error{
			Location: "rate_limit_config_id",
			Message:  "rate limit config not found",
		}).Send(w, r)
		return
	} else {
		indexer.RateLimitConfigId = sql.NullString{
			String: *request.RateLimitConfigId,
			Valid:  true,
		}
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r).WithMessage("Invalid Newznab API key").Send(w, r)
		return
	}

	if err := indexer.Update(); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toNewznabIndexerResponse(indexer))
}

func handleDeleteNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ErrorBadRequest(r).WithMessage("invalid id").Send(w, r)
		return
	}

	existing, err := newznab_indexer.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r).WithMessage("newznab indexer not found").Send(w, r)
		return
	}

	if err := newznab_indexer.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func handleTestNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ErrorBadRequest(r).WithMessage("invalid id").Send(w, r)
		return
	}

	indexer, err := newznab_indexer.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r).WithMessage("newznab indexer not found").Send(w, r)
		return
	}

	if err := indexer.Validate(); err != nil {
		ErrorBadRequest(r).WithMessage("Connection test failed").Send(w, r)
		return
	}

	SendData(w, r, 200, toNewznabIndexerResponse(indexer))
}

func handleToggleNewznabIndexer(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ErrorBadRequest(r).WithMessage("invalid id").Send(w, r)
		return
	}

	indexer, err := newznab_indexer.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if indexer == nil {
		ErrorNotFound(r).WithMessage("newznab indexer not found").Send(w, r)
		return
	}

	if err := newznab_indexer.SetDisabled(id, !indexer.Disabled); err != nil {
		SendError(w, r, err)
		return
	}

	indexer.Disabled = !indexer.Disabled

	SendData(w, r, 200, toNewznabIndexerResponse(indexer))
}

func AddVaultNewznabEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/vault/newznab/indexers", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNewznabIndexers(w, r)
		case http.MethodPost:
			handleCreateNewznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/newznab/indexers/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNewznabIndexer(w, r)
		case http.MethodPatch:
			handleUpdateNewznabIndexer(w, r)
		case http.MethodDelete:
			handleDeleteNewznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/newznab/indexers/{id}/test", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleTestNewznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/newznab/indexers/{id}/toggle", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleToggleNewznabIndexer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
