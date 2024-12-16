package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/stremio/store"
)

func AddStremioEndpoints(mux *http.ServeMux) {
	stremio_store.AddStremioStoreEndpoints(mux)
}
