package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
)

type NZBQueueItemResponse struct {
	Id        string `json:"id"`
	User      string `json:"user"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Category  string `json:"category"`
	Priority  int    `json:"priority"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toNzbQueueItemResponse(entry *nzb_info.JobEntry) NZBQueueItemResponse {
	errMsg := ""
	if len(entry.Error) > 0 {
		errMsg = entry.Error[len(entry.Error)-1]
	}
	return NZBQueueItemResponse{
		Id:        entry.Key,
		User:      entry.Payload.Data.User,
		Name:      entry.Payload.Data.Name,
		URL:       entry.Payload.Data.URL,
		Category:  entry.Payload.Data.Category,
		Priority:  entry.Priority,
		Status:    entry.Status,
		Error:     errMsg,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
	}
}

func handleGetNzbQueueItems(w http.ResponseWriter, r *http.Request) {
	items, err := nzb_info.GetAllJob()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]NZBQueueItemResponse, len(items))
	for i, item := range items {
		data[i] = toNzbQueueItemResponse(&item)
	}

	SendData(w, r, 200, data)
}

func handleGetNzbQueueItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := nzb_info.GetJobById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if item == nil {
		ErrorNotFound(r).WithMessage("queue item not found").Send(w, r)
		return
	}

	SendData(w, r, 200, toNzbQueueItemResponse(item))
}

func handleDeleteNzbQueueItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := nzb_info.GetJobById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r).WithMessage("queue item not found").Send(w, r)
		return
	}

	if err := nzb_info.DeleteJob(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func AddNzbQueueEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/usenet/queue", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNzbQueueItems(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/queue/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNzbQueueItem(w, r)
		case http.MethodDelete:
			handleDeleteNzbQueueItem(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
