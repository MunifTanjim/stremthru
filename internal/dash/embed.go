package dash

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/server"
)

//go:embed fs/**
var spaFS embed.FS

func GetFileHandler() http.Handler {
	dashFS, err := fs.Sub(spaFS, "fs")
	if err != nil {
		panic(err)
	}
	handler := http.FileServerFS(dashFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.NoRequestLog = true

		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			r.URL.Path = "_shell.html"
		}
		handler.ServeHTTP(w, r)
	})
}
