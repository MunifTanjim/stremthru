package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/store"
	"github.com/MunifTanjim/stremthru/internal/stremio/wrap"
)

func AddStremioEndpoints(mux *http.ServeMux) {
	if config.StremioAddon.IsEnabled("store") {
		stremio_store.AddStremioStoreEndpoints(mux)
	}
	if config.StremioAddon.IsEnabled("wrap") {
		stremio_wrap.AddStremioWrapEndpoints(mux)
	}
}
