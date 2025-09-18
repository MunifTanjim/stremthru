package meta_letterboxd

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var SendResponse = shared.SendResponse

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
	})
}

func AddEndpoints(mux *http.ServeMux) {
	if !config.Integration.Letterboxd.IsEnabled() {
		return
	}

	mux.HandleFunc("/v0/meta/lists/letterboxd/{list_id}", handleGetLetterboxdListById)

	router := http.NewServeMux()

	router.HandleFunc("/lists/{list_id}", handleGetLetterboxdListById)
	router.HandleFunc("/users/{user_id}/lists/watchlist", handleGetLetterboxdUserWatchlist)

	mux.Handle("/v0/meta/letterboxd/", http.StripPrefix("/v0/meta/letterboxd", commonMiddleware(router)))
}
