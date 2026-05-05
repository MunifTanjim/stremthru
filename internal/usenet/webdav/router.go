package usenet_webdav

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/util"
	"golang.org/x/net/webdav"
)

func AddEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasNewz() || !config.Feature.HasVault() {
		return
	}

	handler := &webdav.Handler{
		Prefix:     "/v0/webdav/newz/",
		FileSystem: NewFileSystem(),
		LockSystem: webdav.NewMemLS(),
	}

	mux.Handle("/v0/webdav/newz/", withAuth(handler))
}

func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get(server.HEADER_AUTHORIZATION), "Basic "))
		if token == "" {
			w.Header().Set(server.HEADER_WWW_AUTHENTICATE, `Basic realm="stremthru"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		auth, err := util.ParseBasicAuth(token)
		if err != nil || config.Auth.GetPassword(auth.Username) != auth.Password {
			w.Header().Set(server.HEADER_WWW_AUTHENTICATE, `Basic realm="stremthru"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
