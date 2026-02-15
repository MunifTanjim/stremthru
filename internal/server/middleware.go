package server

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

func Middleware(middlewares ...MiddlewareFunc) MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

func extractProxyAuthToken(r *http.Request, readQuery bool) (token string, hasToken bool) {
	token = r.Header.Get(HEADER_STREMTHRU_AUTHORIZATION)
	if token == "" {
		token = r.Header.Get(HEADER_PROXY_AUTHORIZATION)
		if token != "" {
			r.Header.Del(HEADER_PROXY_AUTHORIZATION)
		}
	}
	if token == "" && readQuery {
		token = r.URL.Query().Get("token")
	}
	token = strings.TrimPrefix(token, "Basic ")
	return token, token != ""
}

func GetProxyAuthorization(r *http.Request, readQuery bool) (isAuthorized bool, user, pass string) {
	token, hasToken := extractProxyAuthToken(r, readQuery)
	auth, err := util.ParseBasicAuth(token)
	isAuthorized = hasToken && err == nil && config.UserAuth.GetPassword(auth.Username) == auth.Password
	user = auth.Username
	pass = auth.Password
	return isAuthorized, user, pass
}

func AdminAuthed(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get(HEADER_AUTHORIZATION), "Basic "))
		if token == "" {
			ErrorUnauthorized(r).Send(w, r)
			return
		}
		if auth, err := util.ParseBasicAuth(token); err != nil || config.AdminPassword.GetPassword(auth.Username) != auth.Password {
			ErrorUnauthorized(r).Send(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
