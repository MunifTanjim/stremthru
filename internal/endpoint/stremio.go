package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/disabled"
	stremio_list "github.com/MunifTanjim/stremthru/internal/stremio/list"
	"github.com/MunifTanjim/stremthru/internal/stremio/root"
	"github.com/MunifTanjim/stremthru/internal/stremio/sidekick"
	"github.com/MunifTanjim/stremthru/internal/stremio/store"
	stremio_torz "github.com/MunifTanjim/stremthru/internal/stremio/torz"
	"github.com/MunifTanjim/stremthru/internal/stremio/wrap"
)

func AddStremioEndpoints(mux *http.ServeMux) {
	stremio_root.AddStremioEndpoints(mux)

	if config.Feature.IsEnabled(config.FeatureStremioList) {
		stremio_list.AddEndpoints(mux)
	}
	if config.Feature.IsEnabled(config.FeatureStremioStore) {
		stremio_store.AddStremioStoreEndpoints(mux)
	}
	if config.Feature.IsEnabled(config.FeatureStremioWrap) {
		stremio_wrap.AddStremioWrapEndpoints(mux)
	}
	if config.Feature.IsEnabled(config.FeatureStremioSidekick) {
		stremio_sidekick.AddStremioSidekickEndpoints(mux)
		stremio_disabled.AddStremioDisabledEndpoints(mux)
	}
	if config.Feature.IsEnabled(config.FeatureStremioTorz) {
		stremio_torz.AddStremioTorzEndpoints(mux)
	}
}
