package stremio_list

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/list/configure", http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)
	//
	// router.HandleFunc("/{userData}/stream/{contentType}/{idJson}", withCors(handleStream))
	//
	// router.HandleFunc("/{userData}/_/strem/{stremId}/{storeCode}/{magnetHash}/{fileIdx}/{$}", withCors(handleStrem))
	// router.HandleFunc("/{userData}/_/strem/{stremId}/{storeCode}/{magnetHash}/{fileIdx}/{fileName}", withCors(handleStrem))

	mux.Handle("/stremio/list/", http.StripPrefix("/stremio/list", commonMiddleware(router)))
}
