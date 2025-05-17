package stremio_disabled

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	manifest, err := GetDisabledManifest(r.PathValue("manifestUrl"))
	if err != nil {
		shared.SendError(w, r, err)
		return
	}

	stremio_shared.SendResponse(w, r, 200, manifest)
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	http.Redirect(w, r, "/stremio/sidekick/?addon_operation=manage&try_load_addons=1", http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "manifestUrl")
	})
}

func AddStremioDisabledEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{manifestUrl}/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{manifestUrl}/configure", handleConfigure)

	mux.Handle("/stremio/disabled/", http.StripPrefix("/stremio/disabled", commonMiddleware(router)))
}
