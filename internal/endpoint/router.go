package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/newz"
	"github.com/MunifTanjim/stremthru/internal/sabnzbd"
	"github.com/MunifTanjim/stremthru/internal/torz"
	usenet_webdav "github.com/MunifTanjim/stremthru/internal/usenet/webdav"
)

func AddEndpoints(mux *http.ServeMux) {
	newz.AddEndpoints(mux)
	torz.AddEndpoints(mux)

	sabnzbd.AddEndpoints(mux)

	usenet_webdav.AddEndpoints(mux)
}
