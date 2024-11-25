package context

import (
	"context"
	"net/http"

	"github.com/MunifTanjim/stremthru/store"
)

type requestContextKey struct{}

type RequestContext struct {
	Store             store.Store
	StoreAuthToken    string
	IsProxyAuthorized bool
	ProxyAuthUser     string
	ProxyAuthPassword string
	ClientIP          string // optional
}

func SetRequestContext(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), requestContextKey{}, &RequestContext{})
	return r.WithContext(ctx)
}

func GetRequestContext(r *http.Request) *RequestContext {
	return r.Context().Value(requestContextKey{}).(*RequestContext)
}
