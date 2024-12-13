package endpoint

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

func withRequestContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, context.SetRequestContext(r))
	})
}

func Middleware(middlewares ...MiddlewareFunc) MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return withRequestContext(next)
	}
}

func ProxyAuthContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.GetRequestContext(r)

		token, hasToken := extractProxyAuthToken(r)
		_, username, password, ok := parseBasicAuthToken(token)

		ctx.IsProxyAuthorized = hasToken && ok && config.ProxyAuthPassword.GetPassword(username) == password
		ctx.ProxyAuthUser = username
		ctx.ProxyAuthPassword = password

		next.ServeHTTP(w, r)
	})
}

func ProxyAuthRequired(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.GetRequestContext(r)

		if !ctx.IsProxyAuthorized {
			w.Header().Add("Proxy-Authenticate", "Basic")
			SendError(w, shared.ErrorProxyAuthRequired(r))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getStoreAuthToken(r *http.Request) string {
	authHeader := r.Header.Get("X-StremThru-Store-Authorization")
	if authHeader == "" {
		authHeader = r.Header.Get("Authorization")
	}
	if authHeader == "" {
		ctx := context.GetRequestContext(r)
		if ctx.IsProxyAuthorized && ctx.Store != nil {
			if token := config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(ctx.Store.GetName())); token != "" {
				return token
			}
		}
	}
	_, token, _ := strings.Cut(authHeader, " ")
	return strings.TrimSpace(token)
}

func StoreContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store, err := getStore(r)
		if err != nil {
			SendError(w, err)
			return
		}
		ctx := context.GetRequestContext(r)
		ctx.Store = store
		ctx.StoreAuthToken = getStoreAuthToken(r)
		ctx.PeerToken = r.Header.Get("X-StremThru-Peer-Token")
		if !ctx.IsProxyAuthorized {
			ctx.ClientIP = core.GetClientIP(r)
		}
		w.Header().Add("X-StremThru-Store-Name", r.Header.Get("X-StremThru-Store-Name"))
		next.ServeHTTP(w, r)
	})
}

func StoreRequired(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.GetRequestContext(r)

		if ctx.Store == nil {
			SendError(w, shared.ErrorBadRequest(r, "missing store"))
			return
		}

		if ctx.StoreAuthToken == "" {
			w.Header().Add("WWW-Authenticate", "Bearer realm=\"store:"+string(ctx.Store.GetName())+"\"")
			SendError(w, shared.ErrorUnauthorized(r))
			return
		}

		next.ServeHTTP(w, r)
	})
}
