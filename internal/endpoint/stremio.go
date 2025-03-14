package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/disabled"
	"github.com/MunifTanjim/stremthru/internal/stremio/root"
	"github.com/MunifTanjim/stremthru/internal/stremio/sidekick"
	"github.com/MunifTanjim/stremthru/internal/stremio/store"
	"github.com/MunifTanjim/stremthru/internal/stremio/wrap"
)

func AddStremioEndpoints(mux *http.ServeMux) {
	stremio_root.AddStremioEndpoints(mux)

	if config.StremioAddon.IsEnabled("store") {
		stremio_store.AddStremioStoreEndpoints(mux)
	}
	if config.StremioAddon.IsEnabled("wrap") {
		stremio_wrap.AddStremioWrapEndpoints(mux)
	}
	if config.StremioAddon.IsEnabled("sidekick") {
		stremio_sidekick.AddStremioSidekickEndpoints(mux)
		stremio_disabled.AddStremioDisabledEndpoints(mux)
	}
}
