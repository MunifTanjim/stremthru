package torz

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
	storemiddleware "github.com/MunifTanjim/stremthru/internal/store/middleware"
)

func AddEndpoints(mux *http.ServeMux) {
	withStore := server.Middleware(storemiddleware.WithStoreContext, storemiddleware.RequireStore)

	mux.HandleFunc("/v0/store/torz", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreTorzList(w, r)
		case http.MethodPost:
			handleStoreTorzAdd(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/torz/check", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreTorzCheck(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/torz/{torzId}", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStoreTorzGet(w, r)
		case http.MethodDelete:
			handleStoreTorzRemove(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	mux.HandleFunc("/v0/store/torz/link/generate", withStore(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleStoreTorzLinkGenerate(w, r)
		default:
			server.ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
