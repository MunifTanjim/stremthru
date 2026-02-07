package endpoint

import (
	"errors"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	storemiddleware "github.com/MunifTanjim/stremthru/internal/store/middleware"
	"github.com/MunifTanjim/stremthru/store"
)

func StoreContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := storemiddleware.PrepareStoreContext(w, r); err != nil {
			if errors.Is(err, store.ErrInvalidName) {
				e := core.NewStoreError(err.Error())
				if errors.Is(err, store.ErrInvalidName) {
					e.Code = core.ErrorCodeStoreNameInvalid
				}
			}
			server.SendError(w, r, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireStore(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := storecontext.Get(r)

		if ctx.Store == nil {
			shared.ErrorBadRequest(r, "missing store").Send(w, r)
			return
		}

		if ctx.StoreAuthToken == "" {
			w.Header().Add("WWW-Authenticate", "Bearer realm=\"store:"+string(ctx.Store.GetName())+"\"")
			shared.ErrorUnauthorized(r).Send(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
