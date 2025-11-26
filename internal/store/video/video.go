package store_video

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
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

type StoreVideoName = string

const (
	StoreVideoName200                      StoreVideoName = "200"
	StoreVideoName401                      StoreVideoName = "401"
	StoreVideoName403                      StoreVideoName = "403"
	StoreVideoName429                      StoreVideoName = "429"
	StoreVideoName451                      StoreVideoName = "451"
	StoreVideoName500                      StoreVideoName = "500"
	StoreVideoNameContentProxyLimitReached StoreVideoName = "content_proxy_limit_reached"
	StoreVideoNameDownloadFailed           StoreVideoName = "download_failed"
	StoreVideoNameDownloading              StoreVideoName = "downloading"
	StoreVideoNameNoMatchingFile           StoreVideoName = "no_matching_file"
	StoreVideoNameStoreLimitExceeded       StoreVideoName = "store_limit_exceeded"
	StoreVideoNamePaymentRequired          StoreVideoName = "payment_required"
)

func GetLink(name StoreVideoName, r *http.Request) string {
	return shared.ExtractRequestBaseURL(r).JoinPath("/v0/store/_/static/" + name + ".mp4").String()
}

func Redirect(name StoreVideoName, w http.ResponseWriter, r *http.Request) (url string) {
	url = GetLink(name, r)
	http.Redirect(w, r, url, http.StatusFound)
	return url
}
