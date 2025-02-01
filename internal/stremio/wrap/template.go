package stremio_wrap

import (
	"bytes"
	"html/template"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/internal/stremio/template"
)

func getTemplateData() *TemplateData {
	return &TemplateData{
		Base: Base{
			Title:       "StremThru Wrap",
			Description: "Stremio Addon to Wrap another Addon with StremThru",
			NavTitle:    "Wrap",
		},
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
}

type Base = stremio_template.BaseData

type TemplateData struct {
	Base
	Upstream struct {
		URL            string
		IsConfigurable bool
	}
	HasError struct {
		Upstream bool
	}
	Message struct {
		Upstream string
	}
	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasError.Upstream {
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
