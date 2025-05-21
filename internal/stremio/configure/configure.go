package configure

import (
	"bytes"
	"html/template"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/template"
)

type ConfigType string

const (
	ConfigTypeText     ConfigType = "text"
	ConfigTypeNumber   ConfigType = "number"
	ConfigTypePassword ConfigType = "password"
	ConfigTypeCheckbox ConfigType = "checkbox"
	ConfigTypeSelect   ConfigType = "select"
	ConfigTypeURL      ConfigType = "url"
)

type ConfigAction struct {
	Visible bool
	Label   string
	OnClick template.JS
}

type ConfigOption struct {
	Disabled bool
	Value    string
	Label    string
}

type Config struct {
	Key          string
	Type         ConfigType
	Default      string
	Title        string
	Description  template.HTML
	Options      []ConfigOption
	Required     bool
	Autocomplete string
	Error        string
	Action       ConfigAction
}

type Base = stremio_template.BaseData

type TemplateData struct {
	Base
	Configs     []Config
	Error       string
	ManifestURL string
	Script      template.JS
}

func (td *TemplateData) HasError() bool {
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/configure", func(td *TemplateData) *TemplateData {
		td.Version = config.Version
		return td
	}, template.FuncMap{}, "configure.html", "configure_*.html")
}()

func GetPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "configure.html")
}
