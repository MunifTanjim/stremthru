package stremio_store

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/store/configure", http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioStoreEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/catalog/{contentType}/{idJson}", withCors(handleCatalog))
	router.HandleFunc("/{userData}/catalog/{contentType}/{id}/{extraJson}", withCors(handleCatalog))

	router.HandleFunc("/{userData}/meta/{contentType}/{idJson}", withCors(handleMeta))

	router.HandleFunc("/{userData}/stream/{contentType}/{idJson}", withCors(handleStream))

	router.HandleFunc("/{userData}/_/action/{actionId}", withCors(handleAction))
	router.HandleFunc("/{userData}/_/strem/{videoId}", withCors(handleStrem))

	mux.Handle("/stremio/store/", http.StripPrefix("/stremio/store", commonMiddleware(router)))
}
