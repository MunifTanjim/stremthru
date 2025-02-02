package stremio_sidekick

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	"github.com/MunifTanjim/stremthru/stremio"
)

type Base = stremio_template.BaseData

type TemplateData struct {
	Base
	IsAuthed       bool
	Email          string
	Addons         []stremio.Addon
	AddonOperation string
	AddonError     string
	LastAddonIndex int
	LoginMethod    string
	Login          struct {
		Email    string
		Password string
		Token    string
		Error    struct {
			Email    string
			Password string
			Token    string
		}
	}
	BackupRestore struct {
		AddonsRestoreBlob  string
		LibraryRestoreBlob string
		HasError           struct {
			LibraryRestoreBlob bool
		}
		Message struct {
			LibraryRestoreBlob string
		}
		Error struct {
			AddonsRestoreBlob string
		}
	}
}

func getTemplateData(cookie *CookieValue, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "Stremio Sidekick",
			Description: "Extra Features for Stremio",
			NavTitle:    "Sidekick",
		},
	}
	if cookie != nil && !cookie.IsExpired {
		td.IsAuthed = true
		td.Email = cookie.Email()
	}
	if !td.IsAuthed {
		td.Login.Email = ""
		td.Login.Password = ""
	}

	td.LoginMethod = r.URL.Query().Get("login_method")
	if td.LoginMethod == "" {
		hxCurrUrl := r.Header.Get("hx-current-url")
		if hxCurrUrl != "" {
			if hxUrl, err := url.Parse(hxCurrUrl); err == nil {
				td.LoginMethod = hxUrl.Query().Get("login_method")
			}
		}
	}
	if td.LoginMethod == "" {
		td.LoginMethod = "password"
	}

	td.AddonOperation = r.URL.Query().Get("addon_operation")
	if td.AddonOperation == "" {
		hxCurrUrl := r.Header.Get("hx-current-url")
		if hxCurrUrl != "" {
			if hxUrl, err := url.Parse(hxCurrUrl); err == nil {
				td.AddonOperation = hxUrl.Query().Get("addon_operation")
			}
		}
	}
	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/sidekick", func(td *TemplateData) *TemplateData {
		td.Version = config.Version
		if td.Addons == nil {
			td.Addons = []stremio.Addon{}
		}
		if td.AddonOperation == "" {
			td.AddonOperation = "move"
		}
		td.LastAddonIndex = len(td.Addons) - 1
		return td
	}, template.FuncMap{
		"url_path_escape": func(value string) string {
			return url.PathEscape(value)
		},
		"has_prefix": func(value, prefix string) bool {
			return strings.HasPrefix(value, prefix)
		},
		"get_configure_url": func(value stremio.Addon) string {
			if value.Manifest.BehaviorHints != nil && value.Manifest.BehaviorHints.Configurable {
				return strings.Replace(value.TransportUrl, "/manifest.json", "/configure", 1)
			}
			return ""
		},
	}, "sidekick.html", "sidekick_*.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "sidekick.html")
}
