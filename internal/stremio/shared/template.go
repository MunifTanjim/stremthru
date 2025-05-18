package stremio_shared

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
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

func RedirectToConfigurePage[T any](w http.ResponseWriter, r *http.Request, addon string, ud stremio_userdata.UserData[T], tryInstall bool) {
	url := shared.ExtractRequestBaseURL(r).JoinPath("stremio", addon, ud.GetEncoded(), "configure")
	if tryInstall {
		w.Header().Add("hx-trigger", "try_install")
	}

	if r.Header.Get("hx-request") == "true" {
		w.Header().Add("hx-location", url.String())
		w.WriteHeader(200)
	} else {
		http.Redirect(w, r, url.String(), http.StatusFound)
	}
}
