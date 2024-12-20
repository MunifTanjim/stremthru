package stremio_sidekick

import (
	"bytes"
	"embed"
	"html/template"
	"net/url"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/api"
)

//go:embed index.html addons_section.html
var templateFs embed.FS

type TemplateData struct {
	Title          string
	Version        string
	IsAuthed       bool
	Email          string
	Addons         []stremio_api.Addon
	LastAddonIndex int
}

type TemplateExecutor func(data *TemplateData, name string) (bytes.Buffer, error)

var ExecuteTemplate = func() TemplateExecutor {
	funcMap := template.FuncMap{
		"url_path_escape": func(value string) string {
			return url.PathEscape(value)
		},
	}
	tmpl := template.Must(template.New("stremio/sidekick").Funcs(funcMap).ParseFS(templateFs, "*.html"))

	return func(data *TemplateData, name string) (bytes.Buffer, error) {
		data.Version = config.Version
		if data.Addons == nil {
			data.Addons = []stremio_api.Addon{}
		}
		data.LastAddonIndex = len(data.Addons) - 1
		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, name, data)
		return buf, err
	}
}()

type PageGetter func(data *TemplateData) (bytes.Buffer, error)

var GetPage = func() PageGetter {
	return func(data *TemplateData) (bytes.Buffer, error) {
		return ExecuteTemplate(data, "index.html")
	}
}()
