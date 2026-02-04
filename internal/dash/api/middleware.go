package dash_api

import (
	"context"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/server"
)

type reqCtxtKey struct{}

type ReqCtx struct {
	*server.ReqCtx
	Session  *Session
	ClientIP string
}

func (c *ReqCtx) IsAuthed() bool {
	return c.Session != nil && c.Session.User != ""
}

func GetReqCtx(r *http.Request) *ReqCtx {
	return r.Context().Value(reqCtxtKey{}).(*ReqCtx)
}

func withAPIContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := getSession(w, r)
		if err != nil {
			ErrorInternalServerError(r).WithCause(err).Send(w, r)
			return
		}
		ctx := &ReqCtx{
			Session:  session,
			ClientIP: core.GetRequestIP(r),
			ReqCtx:   server.GetReqCtx(r),
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), reqCtxtKey{}, ctx)))
	})
}

func WithMiddleware(middlewares ...server.MiddlewareFunc) server.MiddlewareFunc {
	return server.Middleware(append([]server.MiddlewareFunc{withAPIContext}, middlewares...)...)
}

func EnsureAuthed(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetReqCtx(r)
		if !ctx.IsAuthed() {
			ErrorUnauthorized(r).Send(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
