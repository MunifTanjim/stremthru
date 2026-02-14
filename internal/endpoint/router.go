package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/newz"
	"github.com/MunifTanjim/stremthru/internal/torz"
)

func AddEndpoints(mux *http.ServeMux) {
	newz.AddEndpoints(mux)
	torz.AddEndpoints(mux)
}
