package meta

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var SendResponse = shared.SendResponse

func handleIdMap(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	idType := IdType(r.PathValue("idType"))
	if !idType.IsValid() {
		shared.ErrorBadRequest(r, "invalid idType").Send(w, r)
		return
	}
	id := r.PathValue("id")

	idMap, err := GetIdMap(idType, id)
	if err != nil {
		if errors.Is(err, ErrorUnsupportedId) {
			shared.ErrorBadRequest(r, ErrorUnsupportedId.Error()).Send(w, r)
			return
		}
		shared.ErrorInternalServerError(r, "").WithCause(err).Send(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(time.Duration(6*time.Hour).Seconds())))
	SendResponse(w, r, 200, idMap, nil)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
	})
}

func AddEndpoints(mux *http.ServeMux) {
	router := http.NewServeMux()

	router.HandleFunc("/id-map/{idType}/{id}", handleIdMap)

	mux.Handle("/v0/meta/", http.StripPrefix("/v0/meta", commonMiddleware(router)))
}
