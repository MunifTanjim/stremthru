package stremio_store

import (
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
)

func getStoreNameConfig() configure.Config {
	options := []configure.ConfigOption{
		configure.ConfigOption{Value: "", Label: "StremThru"},
		configure.ConfigOption{Value: "alldebrid", Label: "AllDebrid"},
		configure.ConfigOption{Value: "debridlink", Label: "DebridLink"},
		configure.ConfigOption{Value: "easydebrid", Label: "EasyDebrid"},
		configure.ConfigOption{Value: "offcloud", Label: "Offcloud"},
		configure.ConfigOption{Value: "pikpak", Label: "PikPak"},
		configure.ConfigOption{Value: "premiumize", Label: "Premiumize"},
		configure.ConfigOption{Value: "realdebrid", Label: "RealDebrid"},
		configure.ConfigOption{Value: "torbox", Label: "TorBox"},
	}
	if !config.ProxyStreamEnabled {
		options[0].Disabled = true
		options[0].Label = ""
	}
	config := configure.Config{
		Key:      "store_name",
		Type:     "select",
		Default:  "",
		Title:    "Store Name",
		Options:  options,
		Required: !config.ProxyStreamEnabled,
	}
	return config
}

func getTemplateData() *configure.TemplateData {
	return &configure.TemplateData{
		Base: configure.Base{
			Title:       "StremThru Store",
			Description: "Stremio Addon for Store Catalog and Search",
			NavTitle:    "Store",
		},
		Configs: []configure.Config{
			getStoreNameConfig(),
			configure.Config{
				Key:         "store_token",
				Type:        "password",
				Default:     "",
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
		},
		Script: configure.GetScriptStoreTokenDescription("store_name", "store_token"),
	}
}
