package stremio_root

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_sidekick "github.com/MunifTanjim/stremthru/internal/stremio/sidekick"
	stremio_store "github.com/MunifTanjim/stremthru/internal/stremio/store"
	stremio_torz "github.com/MunifTanjim/stremthru/internal/stremio/torz"
	stremio_wrap "github.com/MunifTanjim/stremthru/internal/stremio/wrap"
	"github.com/MunifTanjim/stremthru/stremio"
)

func getAddonCatalog(r *http.Request) *stremio.AddonCatalogHandlerResponse {
	addons := []stremio.Addon{}

	if config.Feature.IsEnabled(config.FeatureStremioWrap) {
		addons = append(addons, stremio.Addon{
			Manifest:      *stremio_wrap.GetManifest(r, []stremio.Manifest{}, &stremio_wrap.UserData{}),
			TransportName: "http",
			TransportUrl:  shared.ExtractRequestBaseURL(r).JoinPath("stremio/wrap/manifest.json").String(),
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioStore) {
		addons = append(addons, stremio.Addon{
			Manifest:      *stremio_store.GetManifest(r, &stremio_store.UserData{}),
			TransportName: "http",
			TransportUrl:  shared.ExtractRequestBaseURL(r).JoinPath("stremio/store/manifest.json").String(),
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioTorz) {
		addons = append(addons, stremio.Addon{
			Manifest:      *stremio_torz.GetManifest(r, &stremio_torz.UserData{}),
			TransportName: "http",
			TransportUrl:  shared.ExtractRequestBaseURL(r).JoinPath("stremio/torz/manifest.json").String(),
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioSidekick) {
		addons = append(addons, stremio.Addon{
			Manifest:      *stremio_sidekick.GetManifest(r),
			TransportName: "http",
			TransportUrl:  shared.ExtractRequestBaseURL(r).JoinPath("stremio/sidekick/manifest.json").String(),
		})
	}

	return &stremio.AddonCatalogHandlerResponse{Addons: addons}
}
