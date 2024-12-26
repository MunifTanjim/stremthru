package store_video

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
)

//go:embed *.mp4
var videoFS embed.FS

func Serve(name string, w http.ResponseWriter, r *http.Request) error {
	if !strings.HasSuffix(name, ".mp4") {
		name += ".mp4"
	}

	file, err := videoFS.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(404)
			return nil
		}
		return err
	}
	defer file.Close()

	if f, ok := file.(io.ReadSeeker); ok {
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeContent(w, r, name, config.ServerStartTime, f)
		return nil
	}

	return errors.New("unexpected error from store video")
}
