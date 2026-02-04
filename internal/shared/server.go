package shared

import (
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/rs/xid"
)

func EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var reqLog = logger.Scoped("http")

func RootServerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		ctx := &server.ReqCtx{StartTime: time.Now(), ReqMethod: r.Method, ReqPath: r.URL.Path, ReqQuery: r.URL.Query()}
		r = server.SetReqCtx(r, ctx)

		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				reqLog.Error("panic recovered", "error", err, "stack", string(buf), "req.id", ctx.RequestId)
				ErrorInternalServerError(r, "").Send(rw, r)
				logRequest(rw, r)
			}
		}()

		ctx.RequestId = r.Header.Get("Request-ID")
		if ctx.RequestId == "" {
			ctx.RequestId = xid.New().String()
			r.Header.Set("Request-ID", ctx.RequestId)
		}
		w.Header().Set("Request-ID", ctx.RequestId)

		ctx.Log = logger.New(r.Context(), "req.id", ctx.RequestId)

		next.ServeHTTP(rw, r)
		logRequest(rw, r)
	})
}

type ResponseWriter interface {
	http.ResponseWriter

	getStatusCode() int
}

type responseWriter struct {
	http.ResponseWriter

	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) getStatusCode() int {
	return rw.statusCode
}

func logRequest(w *responseWriter, r *http.Request) {
	ctx := server.GetReqCtx(r)
	if ctx.NoRequestLog {
		return
	}

	status := w.getStatusCode()

	level := slog.LevelError
	if status < 400 {
		level = slog.LevelDebug
	} else if status < 500 {
		level = slog.LevelWarn
	}
	attrs := []slog.Attr{
		slog.Group(
			"req",
			slog.String("id", ctx.RequestId),
			slog.String("method", ctx.ReqMethod),
			slog.String("path", ctx.ReqPath),
			slog.String("query", ctx.ReqQuery.Encode()),
		),
		slog.Int("status", w.getStatusCode()),
		slog.Duration("latency", time.Since(ctx.StartTime)),
	}
	if ctx.Error != nil {
		attrs = append(attrs, slog.Any("error", ctx.Error))
	}
	reqLog.L.LogAttrs(r.Context(), level, "HTTP Request", attrs...)
}
