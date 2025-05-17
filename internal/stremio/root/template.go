package stremio_root

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/template"
)

type Base = stremio_template.BaseData

type templateDataAddon struct {
	Name        string
	Description string
	URL         string
}

type TemplateData struct {
	Base
	Addons      []templateDataAddon
	ManifestURL string
}

func getTemplateData(r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "Stremio Addons",
			Description: "Stremio Addons by StremThru",
			NavTitle:    "Stremio",
		},
		Addons:      []templateDataAddon{},
		ManifestURL: shared.ExtractRequestBaseURL(r).JoinPath("/stremio/manifest.json").String(),
	}

	addons := getAddonCatalog(r).Addons

	for _, addon := range addons {
		td.Addons = append(td.Addons, templateDataAddon{
			Name:        addon.Manifest.Name,
			Description: addon.Manifest.Description,
			URL:         strings.TrimSuffix(addon.TransportUrl, "/manifest.json"),
		})
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/wrap", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		return td
	}, template.FuncMap{}, "root.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "root.html")
}
