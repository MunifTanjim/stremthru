package stremio_newz

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

var IsPublicInstance = config.IsPublicInstance

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/newz/configure", http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.WithCtx(r.Context(), "req.id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
		ctx.RedactURLPathValues(r, "nzbUrl")
	})
}

func AddStremioNewzEndpoints(mux *http.ServeMux) {
	withCors := server.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/stream/{contentType}/{idJson}", withCors(handleStream))

	router.HandleFunc("/{userData}/playback/{stremId}/{mode}/{storeCode}/{nzbUrl}/{$}", withCors(handlePlayback))
	router.HandleFunc("/{userData}/playback/{stremId}/{mode}/{storeCode}/{nzbUrl}/{fileName}", withCors(handlePlayback))

	mux.Handle("/stremio/newz/", http.StripPrefix("/stremio/newz", commonMiddleware(router)))
}
