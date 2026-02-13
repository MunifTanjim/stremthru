package newz

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storemiddleware "github.com/MunifTanjim/stremthru/internal/store/middleware"
)

func AddEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasVault() {
		return
	}

	withStore := server.Middleware(storemiddleware.WithStoreContext, storemiddleware.RequireStore, storemiddleware.EnsureNewzStore)

	mux.HandleFunc("/v0/store/newz", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreNewzList(w, r)
		case http.MethodPost:
			handleStoreNewzAdd(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/newz/check", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreNewzCheck(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/newz/{newzId}", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreNewzGet(w, r)
		case http.MethodDelete:
			handleStoreNewzRemove(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/newz/link/generate", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleStoreNewzLinkGenerate(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/newz/stream/{token}/{filename}", shared.EnableCORS(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreNewzStreamFile(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
