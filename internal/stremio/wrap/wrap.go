package stremio_wrap

import (
	"errors"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/stremio"
)

var IsPublicInstance = config.IsPublicInstance
var MaxPublicInstanceUpstreamCount = 3
var MaxPublicInstanceStoreCount = 3

var addon = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/wrap/configure", http.StatusFound)
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		shared.ErrorBadRequest(r, "failed to get request context: "+err.Error()).Send(w, r)
		return
	}

	manifests, errs := ud.getUpstreamManifests(ctx)
	if errs != nil {
		serr := shared.ErrorInternalServerError(r, "failed to fetch upstream manifests")
		serr.Cause = errors.Join(errs...)
		serr.Send(w, r)
		return
	}

	manifest := GetManifest(r, manifests, ud)

	SendResponse(w, r, 200, manifest)
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	resource := r.PathValue("resource")
	contentType := r.PathValue("contentType")
	id := r.PathValue("id")
	extra := r.PathValue("extra")

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		shared.ErrorBadRequest(r, "failed to get request context: "+err.Error()).Send(w, r)
		return
	}

	switch stremio.ResourceName(resource) {
	case stremio.ResourceNameAddonCatalog:
		ud.fetchAddonCatalog(ctx, w, r, contentType, id)
	case stremio.ResourceNameCatalog:
		ud.fetchCatalog(ctx, w, r, contentType, id, extra)
	case stremio.ResourceNameMeta:
		err = ud.fetchMeta(ctx, w, r, contentType, id, extra)
		if err != nil {
			SendError(w, r, err)
		}
		return
	case stremio.ResourceNameStream:
		res, err := ud.fetchStream(ctx, r, contentType, id)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendResponse(w, r, 200, res)
		return

	case stremio.ResourceNameSubtitles:
		res, err := ud.fetchSubtitles(ctx, contentType, id, extra)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendResponse(w, r, 200, res)
		return
	default:
		addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
			BaseURL:  ud.Upstreams[0].baseUrl,
			Resource: resource,
			Type:     contentType,
			Id:       id,
			Extra:    extra,
			ClientIP: ctx.ClientIP,
		})
	}
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioWrapEndpoints(mux *http.ServeMux) {
	seedDefaultTransformerEntities()

	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/{resource}/{contentType}/{id}", withCors(handleResource))
	router.HandleFunc("/{userData}/{resource}/{contentType}/{id}/{extra}", withCors(handleResource))

	router.HandleFunc("/{userData}/_/strem/{magnetHash}/{fileIdx}/{$}", withCors(handleStrem))
	router.HandleFunc("/{userData}/_/strem/{magnetHash}/{fileIdx}/{fileName}", withCors(handleStrem))

	mux.Handle("/stremio/wrap/", http.StripPrefix("/stremio/wrap", commonMiddleware(router)))
}
