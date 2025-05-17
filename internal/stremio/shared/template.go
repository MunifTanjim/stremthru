package stremio_shared

import (
	"github.com/MunifTanjim/stremthru/internal/config"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
)

func GetStremThruAddons() []stremio_template.BaseDataStremThruAddon {
	addons := []stremio_template.BaseDataStremThruAddon{}

	if config.Feature.IsEnabled(config.FeatureStremioWrap) {
		addons = append(addons, stremio_template.BaseDataStremThruAddon{
			Name: "Wrap",
			URL:  "/stremio/wrap",
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioStore) {
		addons = append(addons, stremio_template.BaseDataStremThruAddon{
			Name: "Store",
			URL:  "/stremio/store",
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioTorz) {
		addons = append(addons, stremio_template.BaseDataStremThruAddon{
			Name: "Torz",
			URL:  "/stremio/torz",
		})
	}
	if config.Feature.IsEnabled(config.FeatureStremioSidekick) {
		addons = append(addons, stremio_template.BaseDataStremThruAddon{
			Name: "Sidekick",
			URL:  "/stremio/sidekick",
		})
	}

	return addons
}
