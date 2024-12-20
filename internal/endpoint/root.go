package endpoint

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
)

//go:embed root.html
var templateBlob string

type rootTemplateDataAddon struct {
	Name string
	URL  string
}

type rootTemplateDataSection struct {
	Title   string        `json:"title"`
	Content template.HTML `json:"content"`
}

type RootTemplateData struct {
	Title       string                    `json:"-"`
	Description template.HTML             `json:"description"`
	Version     string                    `json:"-"`
	Addons      []rootTemplateDataAddon   `json:"-"`
	Sections    []rootTemplateDataSection `json:"sections"`
}

var rootTemplateData = func() RootTemplateData {
	td := RootTemplateData{}
	err := json.Unmarshal([]byte(config.LandingPage), &td)
	if err != nil {
		panic("malformed config for landing page: " + config.LandingPage)
	}
	return td
}()

var ExecuteTemplate = func() func(data *RootTemplateData) (bytes.Buffer, error) {
	tmpl := template.Must(template.New("root.html").Parse(templateBlob))
	return func(data *RootTemplateData) (bytes.Buffer, error) {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		return buf, err
	}
}()

func handleRoot(w http.ResponseWriter, r *http.Request) {
	td := &RootTemplateData{
		Title:       "StremThru",
		Description: rootTemplateData.Description,
		Version:     config.Version,
		Addons:      []rootTemplateDataAddon{},
		Sections:    rootTemplateData.Sections,
	}
	if config.StremioAddon.IsEnabled("store") {
		td.Addons = append(td.Addons, rootTemplateDataAddon{
			Name: "Store",
			URL:  "/stremio/store",
		})
	}
	if config.StremioAddon.IsEnabled("wrap") {
		td.Addons = append(td.Addons, rootTemplateDataAddon{
			Name: "Wrap",
			URL:  "/stremio/wrap",
		})
	}
	if config.StremioAddon.IsEnabled("sidekick") {
		td.Addons = append(td.Addons, rootTemplateDataAddon{
			Name: "Sidekick",
			URL:  "/stremio/sidekick",
		})
	}

	buf, err := ExecuteTemplate(td)
	if err != nil {
		SendError(w, err)
		return
	}
	SendHTML(w, 200, buf)
}

func AddRootEndpoint(mux *http.ServeMux) {
	mux.HandleFunc("/{$}", handleRoot)
}
