package stremio_store

import (
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
)

func getStoreNameConfig() configure.Config {
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
			Description: "Explore and Search Store Catalog",
			NavTitle:    "Store",
		},
		Configs: []configure.Config{
			getStoreNameConfig(),
			{
				Key:         "store_token",
				Type:        "password",
				Default:     "",
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
		},
		Script: configure.GetScriptStoreTokenDescription("'#store_name'", "'#store_token'"),
	}
}
