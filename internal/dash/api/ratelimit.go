package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/ratelimit"
)

type RateLimitConfigResponse struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Limit     int    `json:"limit"`
	Window    string `json:"window"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toRateLimitConfigResponse(item *ratelimit.RateLimitConfig) RateLimitConfigResponse {
	return RateLimitConfigResponse{
		Id:        item.Id,
		Name:      item.Name,
		Limit:     item.Limit,
		Window:    item.Window,
		CreatedAt: item.CAt.Format(time.RFC3339),
		UpdatedAt: item.UAt.Format(time.RFC3339),
	}
}

func handleGetRateLimitConfigs(w http.ResponseWriter, r *http.Request) {
	items, err := ratelimit.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]RateLimitConfigResponse, len(items))
	for i, item := range items {
		data[i] = toRateLimitConfigResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateRateLimitConfigRequest struct {
	Name   string `json:"name"`
	Limit  int    `json:"limit"`
	Window string `json:"window"`
}

func handleCreateRateLimitConfig(w http.ResponseWriter, r *http.Request) {
	request := &CreateRateLimitConfigRequest{}
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
	if request.Limit < 1 {
		errs = append(errs, Error{
			Location: "limit",
			Message:  "limit must be at least 1",
		})
	}
	if request.Window == "" {
		errs = append(errs, Error{
			Location: "window",
			Message:  "missing window",
		})
	} else if _, err := time.ParseDuration(request.Window); err != nil {
		errs = append(errs, Error{
			Location: "window",
			Message:  "invalid duration format (e.g., 30s, 1m, 1h)",
		})
	}

	if len(errs) > 0 {
		ErrorBadRequest(r).Append(errs...).Send(w, r)
		return
	}

	if existing, err := ratelimit.GetByName(request.Name); err != nil {
		SendError(w, r, err)
		return
	} else if existing != nil {
		ErrorBadRequest(r).Append(Error{
			Location: "name",
			Message:  "name already exists",
		}).Send(w, r)
		return
	}

	item, err := ratelimit.Create(request.Name, request.Limit, request.Window)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toRateLimitConfigResponse(item))
}

type UpdateRateLimitConfigRequest struct {
	Name   string `json:"name"`
	Limit  int    `json:"limit"`
	Window string `json:"window"`
}

func handleUpdateRateLimitConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	request := &UpdateRateLimitConfigRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if existing, err := ratelimit.GetById(id); err != nil {
		SendError(w, r, err)
		return
	} else if existing == nil {
		ErrorNotFound(r).WithMessage("rate limit config not found").Send(w, r)
		return
	}

	errs := []Error{}
	if request.Name == "" {
		errs = append(errs, Error{
			Location: "name",
			Message:  "missing name",
		})
	}
	if request.Limit < 1 {
		errs = append(errs, Error{
			Location: "limit",
			Message:  "limit must be at least 1",
		})
	}
	if request.Window == "" {
		errs = append(errs, Error{
			Location: "window",
			Message:  "missing window",
		})
	} else if _, err := time.ParseDuration(request.Window); err != nil {
		errs = append(errs, Error{
			Location: "window",
			Message:  "invalid duration format (e.g., 30s, 1m, 1h)",
		})
	}

	if len(errs) > 0 {
		ErrorBadRequest(r).Append(errs...).Send(w, r)
		return
	}

	if existingByName, err := ratelimit.GetByName(request.Name); err != nil {
		SendError(w, r, err)
		return
	} else if existingByName != nil && existingByName.Id != id {
		ErrorBadRequest(r).Append(Error{
			Location: "name",
			Message:  "name already exists",
		}).Send(w, r)
		return
	}

	item, err := ratelimit.Update(id, request.Name, request.Limit, request.Window)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toRateLimitConfigResponse(item))
}

func handleDeleteRateLimitConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if existing, err := ratelimit.GetById(id); err != nil {
		SendError(w, r, err)
		return
	} else if existing == nil {
		ErrorNotFound(r).WithMessage("rate limit config not found").Send(w, r)
		return
	}

	if err := ratelimit.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func AddRateLimitEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/ratelimit/configs", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetRateLimitConfigs(w, r)
		case http.MethodPost:
			handleCreateRateLimitConfig(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/ratelimit/configs/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			handleUpdateRateLimitConfig(w, r)
		case http.MethodDelete:
			handleDeleteRateLimitConfig(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
