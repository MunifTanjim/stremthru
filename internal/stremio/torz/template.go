package stremio_torz

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type Base = stremio_template.BaseData

type StoreConfig struct {
	Code  stremio_userdata.StoreCode
	Token string
	Error struct {
		Code  string
		Token string
	}
}

type TemplateData struct {
	Base

	Stores           []StoreConfig
	StoreCodeOptions []configure.ConfigOption

	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS

	CanAddStore    bool
	CanRemoveStore bool

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string
}

func (td *TemplateData) HasStoreError() bool {
	for i := range td.Stores {
		if td.Stores[i].Error.Code != "" || td.Stores[i].Error.Token != "" {
			return true
		}
	}
	return false
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasStoreError() {
		return true
	}
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
			Title:       "StremThru Torz",
			Description: "Stremio Addon to access crowdsourced Torz",
			NavTitle:    "Torz",
		},
		Stores:           []StoreConfig{},
		StoreCodeOptions: stremio_shared.GetStoreCodeOptions(),
		Configs: []configure.Config{
			{
				Key:   "cached",
				Type:  configure.ConfigTypeCheckbox,
				Title: "Only Show Cached Content",
			},
		},
		Script: configure.GetScriptStoreTokenDescription("", ""),
	}

	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	for i := range ud.Stores {
		s := &ud.Stores[i]
		td.Stores = append(td.Stores, StoreConfig{
			Code:  s.Code,
			Token: s.Token,
		})
	}

	if len(ud.Stores) == 0 {
		td.Stores = append(td.Stores, StoreConfig{})
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/torz", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.CanAuthorize = !IsPublicInstance
		td.CanAddStore = td.IsAuthed || len(td.Stores) < MaxPublicInstanceStoreCount
		if !IsPublicInstance && td.CanAddStore {
			for i := range td.Stores {
				s := &td.Stores[i]
				if s.Code.IsStremThru() && s.Token != "" {
					td.CanAddStore = false
					td.Stores = td.Stores[i : i+1]
					break
				}
			}
		}
		td.CanRemoveStore = len(td.Stores) > 1

		return td
	}, template.FuncMap{}, "configure_config.html", "torz.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "torz.html")
}
