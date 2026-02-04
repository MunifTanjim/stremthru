package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	trakt_account "github.com/MunifTanjim/stremthru/internal/trakt/account"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type TraktAccountResponse struct {
	Id        string `json:"id"`
	UserName  string `json:"user_name"`
	IsValid   bool   `json:"is_valid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toTraktAccountResponse(item *trakt_account.TraktAccount) TraktAccountResponse {
	username := ""
	if otok := item.OAuthToken(); otok != nil {
		username = otok.UserName
	}
	return TraktAccountResponse{
		Id:        item.Id,
		UserName:  username,
		IsValid:   item.IsValid(),
		CreatedAt: item.CAt.Format(time.RFC3339),
		UpdatedAt: item.UAt.Format(time.RFC3339),
	}
}

func handleGetTraktAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := trakt_account.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]TraktAccountResponse, len(items))
	for i, item := range items {
		data[i] = toTraktAccountResponse(&item)
	}

	SendData(w, r, 200, data)
}

type CreateTraktAccountRequest struct {
	OAuthTokenId string `json:"oauth_token_id"`
}

func handleCreateTraktAccount(w http.ResponseWriter, r *http.Request) {
	request := &CreateTraktAccountRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if request.OAuthTokenId == "" {
		ErrorBadRequest(r).Append(Error{
			Location: "oauth_token_id",
			Message:  "missing oauth_token_id",
		}).Send(w, r)
		return
	}

	account, err := trakt_account.Insert(request.OAuthTokenId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 201, toTraktAccountResponse(account))
}

func handleGetTraktAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	account, err := trakt_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if account == nil {
		ErrorNotFound(r).WithMessage("trakt account not found").Send(w, r)
		return
	}

	forceRefresh := util.StringToBool(r.URL.Query().Get("refresh"), false)
	if forceRefresh {
		client := trakt.GetAPIClient(account.OAuthTokenId)
		_, err := client.RetrieveSettings(&trakt.RetrieveSettingsParams{})
		if err != nil {
			SendError(w, r, err)
			return
		}
	}

	SendData(w, r, 200, toTraktAccountResponse(account))
}

func handleDeleteTraktAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := trakt_account.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r).WithMessage("trakt account not found").Send(w, r)
		return
	}

	if err := trakt_account.Delete(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

type TraktAuthURLResponse struct {
	URL string `json:"url"`
}

func handleGetTraktAuthURL(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	authURL := oauth.TraktOAuthConfig.AuthCodeURL(state)
	SendData(w, r, 200, TraktAuthURLResponse{
		URL: authURL,
	})
}

func AddVaultTraktEndpoints(router *http.ServeMux) {
	if !config.Integration.Trakt.IsEnabled() {
		return
	}

	authed := EnsureAuthed

	router.HandleFunc("/vault/trakt/accounts", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTraktAccounts(w, r)
		case http.MethodPost:
			handleCreateTraktAccount(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/trakt/accounts/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTraktAccount(w, r)
		case http.MethodDelete:
			handleDeleteTraktAccount(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/vault/trakt/auth/url", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTraktAuthURL(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
