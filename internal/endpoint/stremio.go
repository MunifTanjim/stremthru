package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/store"
)

func AddStremioEndpoints(mux *http.ServeMux) {
	if config.StremioAddon.IsEnabled("store") {
		stremio_store.AddStremioStoreEndpoints(mux)
	}
}
