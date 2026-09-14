package endpoint

import (
	"net/http"
	"net/http/pprof"

	"github.com/MunifTanjim/stremthru/internal/server"
)

func AddDebugEndpoints(mux *http.ServeMux) {
	withAdminAuth := server.Middleware(server.AdminAuthed)

	// pprof.Index dispatches the named profiles (heap, goroutine, allocs, ...).
	mux.HandleFunc("/debug/pprof/", withAdminAuth(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", withAdminAuth(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", withAdminAuth(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", withAdminAuth(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", withAdminAuth(pprof.Trace))
}
