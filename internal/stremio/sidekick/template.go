package stremio_sidekick

import (
	"bytes"
	"embed"
	"html/template"
	"net/url"
	"strings"

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
	AddonOperation string
	LastAddonIndex int
}

type TemplateExecutor func(data *TemplateData, name string) (bytes.Buffer, error)

var funcMap = template.FuncMap{
	"url_path_escape": func(value string) string {
		return url.PathEscape(value)
	},
	"has_prefix": func(value, prefix string) bool {
		return strings.HasPrefix(value, prefix)
	},
}

var ExecuteTemplate = func() TemplateExecutor {
	tmpl := template.Must(template.New("stremio/sidekick").Funcs(funcMap).ParseFS(templateFs, "*.html"))
	return func(data *TemplateData, name string) (bytes.Buffer, error) {
		data.Version = config.Version
		if data.Addons == nil {
			data.Addons = []stremio_api.Addon{}
		}
		if data.AddonOperation == "" {
			data.AddonOperation = "move"
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
