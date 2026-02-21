package storemiddleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	"github.com/MunifTanjim/stremthru/store"
)

func getStoreName(r *http.Request, ctx *storecontext.Context) (store.StoreName, error) {
	name := r.Header.Get(server.HEADER_STREMTHRU_STORE_NAME)
	if name == "" {
		if ctx.IsProxyAuthorized {
			name = config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser)
			r.Header.Set(server.HEADER_STREMTHRU_STORE_NAME, name)
		}
	}
	if name == "" {
		return "", nil
	}
	return store.StoreName(name).Validate()
}

func getStore(r *http.Request, ctx *storecontext.Context) (store.Store, error) {
	name, err := getStoreName(r, ctx)
	if err != nil {
		return nil, err
	}
	return shared.GetStore(string(name)), nil
}

func getStoreAuthToken(r *http.Request, ctx *storecontext.Context) string {
	authHeader := r.Header.Get(server.HEADER_STREMTHRU_STORE_AUTHORIZATION)
	if authHeader == "" {
		authHeader = r.Header.Get(server.HEADER_AUTHORIZATION)
	}
	if authHeader == "" {
		if ctx.IsProxyAuthorized && ctx.Store != nil {
			if token := config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(ctx.Store.GetName())); token != "" {
				return token
			}
		}
	}
	_, token, _ := strings.Cut(authHeader, " ")
	return strings.TrimSpace(token)
}

func PrepareStoreContext(w http.ResponseWriter, r *http.Request) error {
	*r = *storecontext.Set(r)

	ctx := storecontext.Get(r)

	ctx.IsProxyAuthorized, ctx.ProxyAuthUser, ctx.ProxyAuthPassword = server.GetProxyAuthorization(r, false)

	store, err := getStore(r, ctx)
	if err != nil {
		return err
	}
	ctx.Store = store
	ctx.StoreAuthToken = getStoreAuthToken(r, ctx)

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	w.Header().Add(server.HEADER_STREMTHRU_STORE_NAME, r.Header.Get(server.HEADER_STREMTHRU_STORE_NAME))
	return nil
}

func WithStoreContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := PrepareStoreContext(w, r); err != nil {
			if errors.Is(err, store.ErrInvalidName) {
				msg := err.Error()
				err = server.ErrorBadRequest(r).
					WithMessage(msg).
					Append(server.Error{
						Domain:       server.ErrorDomainStore,
						LocationType: server.LocationTypeHeader,
						Location:     server.HEADER_STREMTHRU_STORE_NAME,
						Message:      msg,
					})
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
			server.ErrorBadRequest(r).WithMessage("missing store").Send(w, r)
			return
		}

		if ctx.StoreAuthToken == "" {
			w.Header().Add(server.HEADER_WWW_AUTHENTICATE, "Bearer realm=\"store:"+string(ctx.Store.GetName())+"\"")
			server.ErrorUnauthorized(r).Send(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func EnsureNewzStore(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := storecontext.Get(r)

		if _, ok := ctx.Store.(store.NewzStore); !ok {
			server.ErrorBadRequest(r).WithMessage("store does not support newz").Send(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
