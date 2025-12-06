package dash_api

import (
	"errors"
	"net/http"
	"time"

	stremio_account "github.com/MunifTanjim/stremthru/internal/stremio/account"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type StremioAccountResponse struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	IsValid   bool   `json:"is_valid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toStremioAccountResponse(item *stremio_account.StremioAccount) StremioAccountResponse {
	return StremioAccountResponse{
		Id:        item.Id,
		Email:     item.Email,
		IsValid:   item.IsTokenValid(),
		CreatedAt: item.CAt.Format(time.RFC3339),
		UpdatedAt: item.UAt.Format(time.RFC3339),
	}
}

func handleGetStremioAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := stremio_account.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]StremioAccountResponse, len(items))
	for i, item := range items {
		data[i] = toStremioAccountResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateStremioAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleCreateStremioAccount(w http.ResponseWriter, r *http.Request) {
	request := &CreateStremioAccountRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	errs := []Error{}
	if request.Email == "" {
		errs = append(errs, Error{
			Location: "email",
			Message:  "missing email",
		})
	}
	if request.Password == "" {
		errs = append(errs, Error{
			Location: "password",
			Message:  "missing password",
		})
	}
	if len(errs) > 0 {
		ErrorBadRequest(r, "").Append(errs...).Send(w, r)
		return
	}

	existing, err := stremio_account.GetByEmail(request.Email)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing != nil {
		ErrorBadRequest(r, "email already exists").Send(w, r)
		return
	}

	account, err := stremio_account.NewStremioAccount(request.Email, request.Password)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if err := account.Refresh(true); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r, "Invalid Stremio credentials").Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toStremioAccountResponse(account))
}

func handleGetStremioAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	account, err := stremio_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if account == nil {
		ErrorNotFound(r, "stremio account not found").Send(w, r)
		return
	}

	forceRefresh := util.StringToBool(r.URL.Query().Get("refresh"), false)
	if err := account.Refresh(forceRefresh); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r, "Invalid Stremio credentials").Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toStremioAccountResponse(account))
}

type UpdateStremioAccountRequest struct {
	Password string `json:"password"`
}

func handleUpdateStremioAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	request := &UpdateStremioAccountRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if request.Password == "" {
		ErrorBadRequest(r, "").Append(Error{
			Location: "password",
			Message:  "missing password",
		}).Send(w, r)
		return
	}

	account, err := stremio_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if account == nil {
		ErrorNotFound(r, "Account not found").Send(w, r)
		return
	}

	account.SetPassword(request.Password)

	if err := account.Refresh(true); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r, "Invalid Stremio credentials").Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toStremioAccountResponse(account))
}

func handleDeleteStremioAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := stremio_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r, "stremio account not found").Send(w, r)
		return
	}

	if err := stremio_account.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func AddVaultStremioEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/vault/stremio/accounts", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioAccounts(w, r)
		case http.MethodPost:
			handleCreateStremioAccount(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/stremio/accounts/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioAccount(w, r)
		case http.MethodPatch:
			handleUpdateStremioAccount(w, r)
		case http.MethodDelete:
			handleDeleteStremioAccount(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
