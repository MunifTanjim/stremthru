package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger/log"
)

type reqCtxKey struct{}

type ReqCtx struct {
	StartTime    time.Time
	RequestId    string
	Error        error
	ReqMethod    string
	ReqPath      string
	ReqQuery     url.Values
	Log          *log.Logger
	NoRequestLog bool
}

func (ctx *ReqCtx) RedactURLPathValues(r *http.Request, names ...string) {
	for _, name := range names {
		if value := r.PathValue(name); value != "" {
			ctx.ReqPath = strings.Replace(ctx.ReqPath, value, "{"+name+"}", 1)
		}
	}
}

func (ctx *ReqCtx) RedactURLQueryParams(r *http.Request, names ...string) {
	for _, name := range names {
		if _, ok := ctx.ReqQuery[name]; ok {
			ctx.ReqQuery.Set(name, "...redacted...")
		}
	}
}

func SetReqCtx(r *http.Request, reqCtx *ReqCtx) *http.Request {
	ctx := context.WithValue(r.Context(), reqCtxKey{}, reqCtx)
	return r.WithContext(ctx)
}

func GetReqCtx(r *http.Request) *ReqCtx {
	return r.Context().Value(reqCtxKey{}).(*ReqCtx)
}

func GetReqCtxFromContext(ctx context.Context) *ReqCtx {
	reqCtx, ok := ctx.Value(reqCtxKey{}).(*ReqCtx)
	if !ok {
		return nil
	}
	return reqCtx
}
