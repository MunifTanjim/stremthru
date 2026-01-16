package dash_api

import (
	"net/http"
	"time"

	nzb_queue "github.com/MunifTanjim/stremthru/internal/usenet/nzb_queue"
)

type NzbQueueItemResponse struct {
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

func toNzbQueueItemResponse(item *nzb_queue.NzbQueueItem) NzbQueueItemResponse {
	return NzbQueueItemResponse{
		Id:        item.Id,
		User:      item.User,
		Name:      item.Name,
		URL:       item.URL,
		Category:  item.Category,
		Priority:  item.Priority,
		Status:    string(item.Status),
		Error:     item.Error,
		CreatedAt: item.CAt.Format(time.RFC3339),
		UpdatedAt: item.UAt.Format(time.RFC3339),
	}
}

func handleGetNzbQueueItems(w http.ResponseWriter, r *http.Request) {
	items, err := nzb_queue.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]NzbQueueItemResponse, len(items))
	for i, item := range items {
		data[i] = toNzbQueueItemResponse(&item)
	}

	SendData(w, r, 200, data)
}

func handleGetNzbQueueItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := nzb_queue.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if item == nil {
		ErrorNotFound(r, "queue item not found").Send(w, r)
		return
	}

	SendData(w, r, 200, toNzbQueueItemResponse(item))
}

func handleDeleteNzbQueueItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := nzb_queue.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r, "queue item not found").Send(w, r)
		return
	}

	if err := nzb_queue.Delete(id); err != nil {
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
