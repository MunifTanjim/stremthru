package storecontext

import (
	"context"
	"net/http"

	"github.com/MunifTanjim/stremthru/store"
)

type contextKey struct{}

type Context struct {
	Store             store.Store
	StoreAuthToken    string
	IsProxyAuthorized bool
	ProxyAuthUser     string
	ProxyAuthPassword string
	ClientIP          string // optional
}

func Set(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), contextKey{}, &Context{})
	return r.WithContext(ctx)
}

func Get(r *http.Request) *Context {
	return r.Context().Value(contextKey{}).(*Context)
}
