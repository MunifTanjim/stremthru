package stremio_store

import (
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
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
	hideStreamConfig := configure.Config{
		Key:   "hide_stream",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Hide Streams",
	}
	if ud.HideStream {
		hideStreamConfig.Default = "checked"
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
			hideStreamConfig,
		},
		Script: configure.GetScriptStoreTokenDescription("'#store_name'", "'#store_token'"),
	}
}
