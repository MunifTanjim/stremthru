package stremio_store

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

type TemplateData struct {
	Base

	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string

	stremio_userdata.TemplateDataUserData
}

func (td *TemplateData) HasError() bool {
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

func getStoreNameConfig(defaultValue string) configure.Config {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "alldebrid", Label: "AllDebrid"},
		{Value: "debrider", Label: "⚠️ Debrider"},
		{Value: "debridlink", Label: "DebridLink"},
		{Value: "easydebrid", Label: "⚠️ EasyDebrid"},
		{Value: "offcloud", Label: "Offcloud"},
		{Value: "pikpak", Label: "PikPak"},
		{Value: "premiumize", Label: "Premiumize"},
		{Value: "qbittorrent", Label: "qBittorrent"},
		{Value: "realdebrid", Label: "RealDebrid"},
		{Value: "torbox", Label: "TorBox"},
	}
	if config.IsPublicInstance {
		options[0].Disabled = true
		options[0].Label = ""
	}
	conf := configure.Config{
		Key:      "store_name",
		Type:     "select",
		Default:  defaultValue,
		Title:    "Store Name",
		Options:  options,
		Required: config.IsPublicInstance,
	}
	return conf
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	hideCatalogConfig := configure.Config{
		Key:   "hide_catalog",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Hide Catalogs",
	}
	if ud.HideCatalog {
		hideCatalogConfig.Default = "checked"
	}
	hideStreamConfig := configure.Config{
		Key:   "hide_stream",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Hide Streams",
	}
	if ud.HideStream {
		hideStreamConfig.Default = "checked"
	}
	enableWebDLConfig := configure.Config{
		Key:   "enable_webdl",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Enable WebDL",
	}
	if ud.EnableWebDL {
		enableWebDLConfig.Default = "checked"
	}
	enableUsenetConfig := configure.Config{
		Key:   "enable_usenet",
		Type:  configure.ConfigTypeCheckbox,
		Title: "Enable Usenet",
	}
	if ud.EnableUsenet {
		enableUsenetConfig.Default = "checked"
	}

	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Store",
			Description: "Explore and Search Store Catalog",
			NavTitle:    "Store",
		},
		Configs: []configure.Config{
			getStoreNameConfig(ud.StoreName),
			{
				Key:         "store_token",
				Type:        "password",
				Default:     ud.StoreToken,
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
			hideCatalogConfig,
			hideStreamConfig,
			enableWebDLConfig,
			enableUsenetConfig,
		},
		Script: configure.GetScriptStoreTokenDescription("'#store_name'", "'#store_token'"),
	}

	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.Auth.GetPassword(cookie.User()) == cookie.Pass()
	}

	if udManager.IsSaved(ud) {
		td.SavedUserDataKey = udManager.GetId(ud)
	}
	if td.IsAuthed {
		if options, err := stremio_userdata.GetOptions("store"); err != nil {
			LogError(r, "failed to list saved userdata options", err)
		} else {
			td.SavedUserDataOptions = options
		}
	} else if td.SavedUserDataKey != "" {
		if sud, err := stremio_userdata.Get[UserData]("store", td.SavedUserDataKey); err != nil || sud == nil {
			LogError(r, "failed to get saved userdata", err)
		} else {
			td.SavedUserDataOptions = []configure.ConfigOption{{Label: sud.Name, Value: td.SavedUserDataKey}}
		}
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/store", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.IsPublic = config.IsPublicInstance
		td.IsTrusted = config.IsTrusted

		td.CanAuthorize = !config.IsPublicInstance

		td.IsLockedMode = config.Stremio.Locked
		td.IsRedacted = !td.IsAuthed && td.SavedUserDataKey != ""
		if td.IsRedacted {
			redacted := "*******"
			for i := range td.Configs {
				conf := &td.Configs[i]
				if conf.Key == "store_token" && conf.Default != "" {
					conf.Default = redacted
				}
			}
		}

		return td
	}, template.FuncMap{}, "configure_config.html", "configure_submit_button.html", "saved_userdata_field.html", "store.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "store.html")
}

func sendPage(w http.ResponseWriter, r *http.Request, td *TemplateData) {
	page, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, page)
}
