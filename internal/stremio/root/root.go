package stremio_root

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	td := getTemplateData(r)

	page, err := getPage(td)
	if err != nil {
		shared.SendError(w, r, err)
		return
	}
	stremio_shared.SendHTML(w, 200, page)
	return
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	manifest := getManifest(r)

	stremio_shared.SendResponse(w, r, 200, manifest)
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	http.Redirect(w, r, "/stremio/", http.StatusFound)
}

func handleAddonCatalog(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	res := getAddonCatalog(r)

	stremio_shared.SendResponse(w, r, 200, res)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "transportUrl")
	})
}

func AddStremioEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)
	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/addon_catalog/all/stremthru.json", withCors(handleAddonCatalog))
	router.HandleFunc("/configure", handleConfigure)

	mux.Handle("/stremio/", http.StripPrefix("/stremio", commonMiddleware(router)))
}
