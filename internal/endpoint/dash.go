package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/dash"
)

func AddDashEndpoint(mux *http.ServeMux) {
	dash.AddEndpoints(mux)
}
