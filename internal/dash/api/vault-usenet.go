package dash_api

import (
	"context"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/nntp"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	usenet_server "github.com/MunifTanjim/stremthru/internal/usenet/server"
)

type UsenetServerResponse struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	TLS            bool   `json:"tls"`
	TLSSkipVerify  bool   `json:"tls_skip_verify"`
	Priority       int    `json:"priority"`
	IsBackup       bool   `json:"is_backup"`
	MaxConnections int    `json:"max_connections"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func toUsenetServerResponse(item *usenet_server.UsenetServer) UsenetServerResponse {
	return UsenetServerResponse{
		Id:             item.Id,
		Name:           item.Name,
		Host:           item.Host,
		Port:           item.Port,
		Username:       item.Username,
		TLS:            item.TLS,
		TLSSkipVerify:  item.TLSSkipVerify,
		Priority:       item.Priority,
		IsBackup:       item.IsBackup,
		MaxConnections: item.MaxConnections,
		CreatedAt:      item.CAt.Format(time.RFC3339),
		UpdatedAt:      item.UAt.Format(time.RFC3339),
	}
}

func handleGetUsenetServers(w http.ResponseWriter, r *http.Request) {
	items, err := usenet_server.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]UsenetServerResponse, len(items))
	for i, item := range items {
		data[i] = toUsenetServerResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateUsenetServerRequest struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	TLS            bool   `json:"tls"`
	TLSSkipVerify  bool   `json:"tls_skip_verify"`
	Priority       int    `json:"priority"`
	IsBackup       bool   `json:"is_backup"`
	MaxConnections int    `json:"max_connections"`
}

func handleCreateUsenetServer(w http.ResponseWriter, r *http.Request) {
	request := &CreateUsenetServerRequest{}
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
	if request.Host == "" {
		errs = append(errs, Error{
			Location: "host",
			Message:  "missing host",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r, "").Append(errs...).Send(w, r)
		return
	}

	if request.Port == 0 {
		if request.TLS {
			request.Port = 563
		} else {
			request.Port = 119
		}
	}

	if request.MaxConnections == 0 {
		request.MaxConnections = 10
	}

	server, err := usenet_server.NewUsenetServer(
		request.Name,
		request.Host,
		request.Port,
		request.Username,
		request.Password,
		request.TLS,
		request.TLSSkipVerify,
		request.Priority,
		request.IsBackup,
		request.MaxConnections,
	)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if err := server.Upsert(); err != nil {
		SendError(w, r, err)
		return
	}

	usenetmanager.AddServer(server.ProviderId())

	SendData(w, r, 201, toUsenetServerResponse(server))
}

func handleGetUsenetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	server, err := usenet_server.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if server == nil {
		ErrorNotFound(r, "usenet server not found").Send(w, r)
		return
	}

	SendData(w, r, 200, toUsenetServerResponse(server))
}

type UpdateUsenetServerRequest struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	TLS            *bool  `json:"tls"`
	TLSSkipVerify  *bool  `json:"tls_skip_verify"`
	Priority       *int   `json:"priority"`
	IsBackup       *bool  `json:"is_backup"`
	MaxConnections *int   `json:"max_connections"`
}

func handleUpdateUsenetServer(w http.ResponseWriter, r *http.Request) {
	server, err := usenet_server.GetById(r.PathValue("id"))
	if err != nil {
		SendError(w, r, err)
		return
	}
	if server == nil {
		ErrorNotFound(r, "server not found").Send(w, r)
		return
	}

	oldProviderId := server.ProviderId()

	if err := usenetmanager.LockServer(oldProviderId); err != nil {
		ErrorLocked(r, "cannot modify server with active connections").Send(w, r)
		return
	}
	defer usenetmanager.UnlockServer(oldProviderId)

	request := &UpdateUsenetServerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if request.Name != "" {
		server.Name = request.Name
	}
	if request.Host != "" {
		server.Host = request.Host
	}
	if request.Port != 0 {
		server.Port = request.Port
	}
	if request.Username != "" {
		server.Username = request.Username
	}
	if request.Password != "" {
		if err := server.SetPassword(request.Password); err != nil {
			SendError(w, r, err)
			return
		}
	}
	if request.TLS != nil {
		server.TLS = *request.TLS
	}
	if request.TLSSkipVerify != nil {
		server.TLSSkipVerify = *request.TLSSkipVerify
	}
	if request.Priority != nil {
		server.Priority = *request.Priority
	}
	if request.IsBackup != nil {
		server.IsBackup = *request.IsBackup
	}
	if request.MaxConnections != nil {
		server.MaxConnections = *request.MaxConnections
		if server.MaxConnections == 0 {
			server.MaxConnections = 10
		}
	}

	newProviderId := server.ProviderId()

	if err := server.Upsert(); err != nil {
		SendError(w, r, err)
		return
	}

	usenetmanager.UpdateServer(oldProviderId, newProviderId)

	SendData(w, r, 200, toUsenetServerResponse(server))
}

func handleDeleteUsenetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := usenet_server.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r, "usenet server not found").Send(w, r)
		return
	}

	providerId := existing.ProviderId()

	if err := usenetmanager.LockServer(providerId); err != nil {
		ErrorLocked(r, "cannot delete server with active connections").Send(w, r)
		return
	}
	defer usenetmanager.UnlockServer(providerId)

	if err := usenet_server.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	usenetmanager.RemoveServer(providerId)

	SendData(w, r, 204, nil)
}

type PingUsenetServerRequest struct {
	Id            string `json:"id,omitempty"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	TLS           bool   `json:"tls"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

type PingUsenetServerResponse struct {
	Message string `json:"message"`
}

func handlePingUsenetServer(w http.ResponseWriter, r *http.Request) {
	request := &PingUsenetServerRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if request.Id != "" {
		existing, err := usenet_server.GetById(request.Id)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if existing == nil {
			ErrorNotFound(r, "usenet server not found").Send(w, r)
			return
		}
		if request.Password == "" {
			password, err := existing.GetPassword()
			if err != nil {
				SendError(w, r, err)
				return
			}
			request.Password = password
		}
		if request.Port == 0 {
			request.Port = existing.Port
		}
	}

	if request.Port == 0 {
		if request.TLS {
			request.Port = 563
		} else {
			request.Port = 119
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	conn := nntp.Connection{}
	defer conn.Close()

	err := conn.Connect(ctx, &nntp.ConnectionConfig{
		Host:          request.Host,
		Port:          request.Port,
		Username:      request.Username,
		Password:      request.Password,
		TLS:           request.TLS,
		TLSSkipVerify: request.TLSSkipVerify,
		Deadline:      time.Now().Add(15 * time.Second),
		DialTimeout:   10 * time.Second,
		KeepAliveTime: 20 * time.Second,
	})
	if err != nil {
		if nntpErr, ok := err.(*nntp.Error); ok {
			ErrorBadRequest(r, nntpErr.Error()).Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, PingUsenetServerResponse{
		Message: "Connection Successful!",
	})
}

func AddVaultUsenetEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/vault/usenet/servers", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetUsenetServers(w, r)
		case http.MethodPost:
			handleCreateUsenetServer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/usenet/servers/ping", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlePingUsenetServer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/usenet/servers/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetUsenetServer(w, r)
		case http.MethodPatch:
			handleUpdateUsenetServer(w, r)
		case http.MethodDelete:
			handleDeleteUsenetServer(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
