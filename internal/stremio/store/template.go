package stremio_store

import (
	"bytes"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func getStoreNameConfig(defaultValue string) configure.Config {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "alldebrid", Label: "AllDebrid"},
		{Value: "debridlink", Label: "DebridLink"},
		{Value: "easydebrid", Label: "EasyDebrid"},
		{Value: "offcloud", Label: "Offcloud"},
		{Value: "pikpak", Label: "PikPak"},
		{Value: "premiumize", Label: "Premiumize"},
		{Value: "realdebrid", Label: "RealDebrid"},
		{Value: "torbox", Label: "TorBox"},
	}
	if config.IsPublicInstance {
		options[0].Disabled = true
		options[0].Label = ""
	}
	config := configure.Config{
		Key:      "store_name",
		Type:     "select",
		Default:  defaultValue,
		Title:    "Store Name",
		Options:  options,
		Required: config.IsPublicInstance,
	}
	return config
}

func getTemplateData(ud *UserData) *configure.TemplateData {
	hideCatalogConfig := configure.Config{
		Key:   "hide_catalog",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Hide Catalogs",
	}
	if ud.HideCatalog {
		hideCatalogConfig.Default = "checked"
	}
	hideStreamConfig := configure.Config{
		Key:   "hide_stream",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Hide Streams",
	}
	if ud.HideStream {
		hideStreamConfig.Default = "checked"
	}
	enableWebDLConfig := configure.Config{
		Key:   "enable_webdl",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Enable WebDL",
	}
	if ud.EnableWebDL {
		enableWebDLConfig.Default = "checked"
	}
	return &configure.TemplateData{
		Base: configure.Base{
			Title:       "StremThru Store",
			Description: "Explore and Search Store Catalog",
			NavTitle:    "Store",
		},
		Configs: []configure.Config{
			getStoreNameConfig(ud.StoreName),
			{
				Key:         "store_token",
				Type:        "password",
				Default:     ud.StoreToken,
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
			hideCatalogConfig,
			hideStreamConfig,
			enableWebDLConfig,
		},
		Script: configure.GetScriptStoreTokenDescription("'#store_name'", "'#store_token'"),
	}
}

func getPage(td *configure.TemplateData) (bytes.Buffer, error) {
	td.StremThruAddons = stremio_shared.GetStremThruAddons()
	return configure.GetPage(td)
}
