package dash

import (
	"net/http"
	"net/http/httputil"

	"github.com/MunifTanjim/stremthru/internal/config"
	dash_api "github.com/MunifTanjim/stremthru/internal/dash/api"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func commonMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := dash_api.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
	})
}

func AddEndpoints(mux *http.ServeMux) {
	router := http.NewServeMux()

	authed := dash_api.EnsureAuthed

	router.HandleFunc("/auth/signin", dash_api.HandleSignIn)
	router.HandleFunc("/auth/signout", authed(dash_api.HandleSignOut))
	router.HandleFunc("/auth/user", authed(dash_api.HandleGetUser))

	router.HandleFunc("/stats/lists", authed(dash_api.HandleGetListsStats))
	router.HandleFunc("/stats/torrents", authed(dash_api.HandleGetTorrentsStats))
	router.HandleFunc("/stats/server", authed(dash_api.HandleGetServerStats))

	router.HandleFunc("/workers/details", authed(dash_api.HandleGetWorkersDetails))
	router.HandleFunc("/workers/{id}/job-logs", authed(dash_api.HandleGetWorkerJobLogs))

	mux.Handle("/dash/api/", http.StripPrefix("/dash/api", dash_api.WithMiddleware(commonMiddleware)(router.ServeHTTP)))

	switch config.Environment {
	case config.EnvDev:
		handler := httputil.NewSingleHostReverseProxy(util.MustParseURL("http://localhost:3000"))
		mux.Handle("/dash/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := server.GetReqCtx(r)
			ctx.NoRequestLog = true

			handler.ServeHTTP(w, r)
		}))

	case config.EnvProd:
		handler := GetFileHandler()
		mux.Handle("/dash/", http.StripPrefix("/dash", handler))
	}
}
