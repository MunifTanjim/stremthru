package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/newz"
)

func AddEndpoints(mux *http.ServeMux) {
	newz.AddEndpoints(mux)
}
