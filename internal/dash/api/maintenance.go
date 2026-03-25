package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/server"
)

type GetMaintenanceStatusData struct {
	IsActive bool   `json:"is_active"`
	EndsAt   string `json:"ends_at"`
}

func handleGetMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	data := GetMaintenanceStatusData{
		IsActive: server.IsMaintenanceActive(),
	}
	if data.IsActive {
		if endsAt := server.GetMaintenanceEndTime(); !endsAt.IsZero() {
			data.EndsAt = endsAt.Format(time.RFC3339)
		}
	}
	SendData(w, r, 200, data)
}

type StartMaintenanceRequest struct {
	Duration string `json:"duration"`
}

func handleActivateMaintenance(w http.ResponseWriter, r *http.Request) {
	request := &StartMaintenanceRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	duration := 1 * time.Minute
	if request.Duration != "" {
		d, err := time.ParseDuration(request.Duration)
		if err != nil {
			ErrorBadRequest(r).Append(Error{
				Location:     "duration",
				LocationType: server.LocationTypeBody,
				Message:      "invalid duration format",
			}).Send(w, r)
			return
		}
		duration = d
	}

	server.ActivateMaintenance(duration)

	data := GetMaintenanceStatusData{
		IsActive: server.IsMaintenanceActive(),
	}
	if data.IsActive {
		if endsAt := server.GetMaintenanceEndTime(); !endsAt.IsZero() {
			data.EndsAt = endsAt.Format(time.RFC3339)
		}
	}
	SendData(w, r, 200, data)
}

func handleDeactivateMaintenance(w http.ResponseWriter, r *http.Request) {
	server.DeactivateMaintenance()
	SendData(w, r, 200, GetMaintenanceStatusData{
		IsActive: false,
	})
}

func AddMaintenanceEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/maintenance", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetMaintenanceStatus(w, r)
		case http.MethodPost:
			handleActivateMaintenance(w, r)
		case http.MethodDelete:
			handleDeactivateMaintenance(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
