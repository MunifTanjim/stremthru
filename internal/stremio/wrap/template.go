package stremio_wrap

import (
	"bytes"
	"html/template"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/internal/stremio/template"
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
		Key:      "store",
		Type:     "select",
		Default:  "",
		Title:    "Store Name",
		Options:  options,
		Required: !config.ProxyStreamEnabled,
	}
	return config
}

func getTemplateData(ud *UserData) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Wrap",
			Description: "Stremio Addon to Wrap another Addon with StremThru",
			NavTitle:    "Wrap",
		},
		Upstreams: []UpstreamAddon{},
		Configs: []configure.Config{
			getStoreNameConfig(),
			{
				Key:         "token",
				Type:        configure.ConfigTypePassword,
				Default:     "",
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
			{
				Key:     "cached",
				Type:    configure.ConfigTypeCheckbox,
				Title:   "Only Show Cached Content",
				Options: []configure.ConfigOption{},
			},
		},
		Script: configure.GetScriptStoreTokenDescription("store", "token"),
	}
	for _, up := range ud.Upstreams {
		td.Upstreams = append(td.Upstreams, UpstreamAddon{
			URL: up.URL,
		})
	}
	if len(td.Upstreams) == 0 {
		td.Upstreams = append(td.Upstreams, UpstreamAddon{URL: ""})
	}
	return td
}

type Base = stremio_template.BaseData

type UpstreamAddon struct {
	URL            string
	IsConfigurable bool
	Error          string
}

type TemplateData struct {
	Base
	Upstreams   []UpstreamAddon
	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS
}

func (td *TemplateData) HasUpstreamError() bool {
	for i := range td.Upstreams {
		if td.Upstreams[i].Error != "" {
			return true
		}
	}
	return false
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasUpstreamError() {
		return true
	}
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/wrap", func(td *TemplateData) *TemplateData {
		td.Version = config.Version
		return td
	}, template.FuncMap{}, "configure_config.html", "wrap.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "wrap.html")
}
