package dash_api

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"time"

	stremio_account "github.com/MunifTanjim/stremthru/internal/stremio/account"
	stremio_api "github.com/MunifTanjim/stremthru/internal/stremio/api"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	stremio_userdata_account "github.com/MunifTanjim/stremthru/internal/stremio/userdata/account"
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
		ErrorBadRequest(r).Append(errs...).Send(w, r)
		return
	}

	existing, err := stremio_account.GetByEmail(request.Email)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing != nil {
		ErrorBadRequest(r).WithMessage("email already exists").Send(w, r)
		return
	}

	account, err := stremio_account.NewStremioAccount(request.Email, request.Password)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if err := account.Refresh(true); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r).WithMessage("Invalid Stremio credentials").Send(w, r)
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
		ErrorNotFound(r).WithMessage("stremio account not found").Send(w, r)
		return
	}

	forceRefresh := util.StringToBool(r.URL.Query().Get("refresh"), false)
	if err := account.Refresh(forceRefresh); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r).WithMessage("Invalid Stremio credentials").Send(w, r)
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
		ErrorBadRequest(r).Append(Error{
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
		ErrorNotFound(r).WithMessage("Account not found").Send(w, r)
		return
	}

	account.SetPassword(request.Password)

	if err := account.Refresh(true); err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r).WithMessage("Invalid Stremio credentials").Send(w, r)
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
		ErrorNotFound(r).WithMessage("stremio account not found").Send(w, r)
		return
	}

	if err := stremio_account.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

type StremioAccountUserdataResponse struct {
	Addon     string `json:"addon"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func handleGetStremioAccountUserdata(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	account, err := stremio_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if account == nil {
		ErrorNotFound(r).WithMessage("stremio account not found").Send(w, r)
		return
	}

	items, err := stremio_userdata.GetLinkedUserdataByAccountId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]StremioAccountUserdataResponse, len(items))
	for i, item := range items {
		data[i] = StremioAccountUserdataResponse{
			Addon:     item.Addon,
			Key:       item.Key,
			Name:      item.Name,
			CreatedAt: item.CAt.Format(time.RFC3339),
		}
	}

	SendData(w, r, 200, data)
}

var stremioClient = stremio_api.NewClient(&stremio_api.ClientConfig{})

var stremthruSavedUserdataPattern = regexp.MustCompile(`^/stremio/(store|wrap|list|torz)/k\.([a-f0-9]+)/`)

func handleSyncStremioAccountUserdata(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	account, err := stremio_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if account == nil {
		ErrorNotFound(r).WithMessage("stremio account not found").Send(w, r)
		return
	}

	token, err := account.GetValidToken()
	if err != nil {
		if errors.Is(err, stremio_account.ErrorInvalidCredentials) {
			ErrorBadRequest(r).WithMessage("Invalid Stremio credentials").Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	params := &stremio_api.GetAddonsParams{}
	params.APIKey = token
	res, err := stremioClient.GetAddons(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	linked := []StremioAccountUserdataResponse{}
	for _, addon := range res.Data.Addons {
		transportUrl, err := url.Parse(addon.TransportUrl)
		if err != nil {
			continue
		}

		matches := stremthruSavedUserdataPattern.FindStringSubmatch(transportUrl.Path)
		if len(matches) < 3 {
			continue
		}

		addonName := matches[1]
		userdataKey := matches[2]

		userdata, err := stremio_userdata.Get[any](addonName, userdataKey)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if userdata == nil {
			continue
		}

		if err := stremio_userdata_account.Link(addonName, userdataKey, id); err != nil {
			SendError(w, r, err)
			return
		}

		linked = append(linked, StremioAccountUserdataResponse{
			Addon:     userdata.Addon,
			Key:       userdata.Key,
			Name:      userdata.Name,
			CreatedAt: time.Now().Format(time.RFC3339),
		})
	}

	SendData(w, r, 200, linked)
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
	router.HandleFunc("/vault/stremio/accounts/{id}/userdata", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetStremioAccountUserdata(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/stremio/accounts/{id}/userdata/sync", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleSyncStremioAccountUserdata(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
