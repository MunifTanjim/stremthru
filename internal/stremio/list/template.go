package stremio_list

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
)

var IsPublicInstance = config.IsPublicInstance

type Base = stremio_template.BaseData

type TemplateData struct {
	Base

	Configs     []configure.Config
	ManifestURL string
	Script      template.JS

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string
}

func (td *TemplateData) HasFieldError() bool {
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru List",
			Description: "Stremio Addon to access various Lists",
			NavTitle:    "List",
		},
		Configs: []configure.Config{
			{
				Key:         "mdblist_api_key",
				Type:        configure.ConfigTypeText,
				Title:       "MDBList API Key",
				Default:     ud.MDBListAPIkey,
				Description: `<a href="https://mdblist.com/preferences" target="_blank">API Key</a>`,
			},
		},
	}

	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/list", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.CanAuthorize = !IsPublicInstance

		return td
	}, template.FuncMap{}, "configure_config.html", "list.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "list.html")
}
