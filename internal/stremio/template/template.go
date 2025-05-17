package stremio_template

import (
	"bytes"
	"embed"
	"html/template"
)

type BaseDataStremThruAddon struct {
	Name string
	URL  string
}

type BaseData struct {
	Title           string
	Description     string
	NavTitle        string
	Version         string
	StremThruAddons []BaseDataStremThruAddon
}

type Executor[T any] func(data *T, name string) (bytes.Buffer, error)

//go:embed *.html
var templateFs embed.FS

func GetExecutor[T any](name string, prepare func(data *T) *T, funcMap template.FuncMap, patterns ...string) Executor[T] {
	patterns = append(patterns, "layout.html")
	tmpl := template.Must(template.New("stremio").Funcs(funcMap).ParseFS(templateFs, patterns...))
	return func(data *T, name string) (bytes.Buffer, error) {
		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, name, prepare(data))
		return buf, err
	}
}
