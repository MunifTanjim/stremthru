package stremio_wrap

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/internal/stremio/template"
)

func getStoreNameConfig() configure.Config {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "alldebrid", Label: "AllDebrid"},
		{Value: "debridlink", Label: "DebridLink"},
		{Value: "easydebrid", Label: "EasyDebrid"},
		{Value: "offcloud", Label: "Offcloud"},
		{Value: "pikpak", Label: "PikPak"},
		{Value: "premiumize", Label: "Premiumize"},
		{Value: "realdebrid", Label: "RealDebrid"},
		{Value: "torbox", Label: "TorBox"},
	}
	if !config.ProxyStreamEnabled {
		options[0].Disabled = true
		options[0].Label = ""
	}
	config := configure.Config{
		Key:      "store",
		Type:     "select",
		Default:  "",
		Title:    "Store Name",
		Options:  options,
		Required: !config.ProxyStreamEnabled,
	}
	return config
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Wrap",
			Description: "Stremio Addon to Wrap another Addon with StremThru",
			NavTitle:    "Wrap",
		},
		Upstreams: []UpstreamAddon{},
		Configs: []configure.Config{
			getStoreNameConfig(),
			{
				Key:         "token",
				Type:        configure.ConfigTypePassword,
				Default:     "",
				Title:       "Store Token",
				Description: "",
				Required:    true,
			},
			{
				Key:     "cached",
				Type:    configure.ConfigTypeCheckbox,
				Title:   "Only Show Cached Content",
				Options: []configure.ConfigOption{},
			},
		},
		Script: configure.GetScriptStoreTokenDescription("store", "token"),

		ExtractorIds: []string{},
		TemplateIds:  []string{},
	}

	if cookie, err := getCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	isExecutingAction := r.Header.Get("x-addon-configure-action") != ""

	td.TemplateId = ud.TemplateId
	td.Template = ud.template
	if !isExecutingAction {
		if td.TemplateId != "" {
			var storedBlob StreamTransformerTemplateBlob
			if err := templateStore.Get(td.TemplateId, &storedBlob); err == nil {
				if !storedBlob.IsEmpty() {
					if storedBlob.Name != td.Template.Name {
						td.TemplateError.Name = "Template is not updated"
					} else if storedBlob.Description != td.Template.Description {
						td.TemplateError.Description = "Template is not updated"
					}
				} else {
					td.TemplateError.Name = "Template is not saved"
					td.TemplateError.Description = "Template is not saved"
				}
			}
		} else if !td.Template.IsEmpty() {
			td.TemplateError.Name = "Template is not saved"
			td.TemplateError.Description = "Template is not saved"
		}

		if td.TemplateError.IsEmpty() && !td.Template.IsEmpty() {
			if t, err := td.Template.Parse(); err != nil {
				if t.Name == nil {
					td.TemplateError.Name = err.Error()
				} else {
					td.TemplateError.Description = err.Error()
				}
			}
		}
	}

	shouldHaveExtractor := !td.Template.IsEmpty() || td.TemplateId != ""

	for _, up := range ud.Upstreams {
		extractorError := ""
		if !isExecutingAction {
			if up.ExtractorId != "" {
				var storedBlob StreamTransformerExtractorBlob
				if err := extractorStore.Get(up.ExtractorId, &storedBlob); err == nil {
					if storedBlob != "" {
						if storedBlob != up.extractor {
							extractorError = "Extractor is not updated"
						}
					} else {
						extractorError = "Extractor is not saved"
					}
				}
			} else if up.extractor != "" {
				extractorError = "Extractor is not saved"
			}

			if up.ExtractorId != "" || up.extractor != "" {
				shouldHaveExtractor = true
			}

			if extractorError == "" && up.extractor != "" {
				if _, err := up.extractor.Parse(); err != nil {
					extractorError = err.Error()
				}
			}
		}
		td.Upstreams = append(td.Upstreams, UpstreamAddon{
			URL:            up.URL,
			ExtractorId:    up.ExtractorId,
			Extractor:      up.extractor,
			ExtractorError: extractorError,
		})
	}

	if len(td.Upstreams) == 0 {
		td.Upstreams = append(td.Upstreams, UpstreamAddon{URL: ""})
	}

	if shouldHaveExtractor {
		for i := range td.Upstreams {
			up := &td.Upstreams[i]
			if up.ExtractorId == "" && up.Extractor == "" {
				up.ExtractorError = "Extractor is missing"
			}
		}

		if td.TemplateId == "" && td.Template.IsEmpty() {
			td.TemplateError.Name = "Template is missing"
			td.TemplateError.Description = "Template is missing"
		}
	}

	extractors, err := extractorStore.List()
	if err != nil {
		core.LogError("[stremio/wrap] failed to list extractors", err)
	} else {
		extractorIds := make([]string, len(extractors))
		for i, extractor := range extractors {
			extractorIds[i] = extractor.Key
		}
		td.ExtractorIds = extractorIds
	}

	templates, err := templateStore.List()
	if err != nil {
		core.LogError("[stremio/wrap] failed to list templates", err)
	} else {
		templateIds := make([]string, len(templates))
		for i, template := range templates {
			templateIds[i] = template.Key
		}
		td.TemplateIds = templateIds
	}

	return td
}

type Base = stremio_template.BaseData

type UpstreamAddon struct {
	URL            string
	IsConfigurable bool
	Error          string
	ExtractorId    string
	Extractor      StreamTransformerExtractorBlob
	ExtractorError string
}

type TemplateData struct {
	Base
	Upstreams   []UpstreamAddon
	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS

	SupportAdvanced bool
	IsAuthed        bool
	ExtractorIds    []string
	TemplateIds     []string
	TemplateId      string
	Template        StreamTransformerTemplateBlob
	TemplateError   StreamTransformerTemplateBlob
}

func (td *TemplateData) HasUpstreamError() bool {
	for i := range td.Upstreams {
		if td.Upstreams[i].Error != "" || td.Upstreams[i].ExtractorError != "" {
			return true
		}
	}
	return false
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasUpstreamError() {
		return true
	}
	if td.TemplateError.Name != "" || td.TemplateError.Description != "" {
		return true
	}
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/wrap", func(td *TemplateData) *TemplateData {
		td.Version = config.Version
		td.SupportAdvanced = SupportAdvanced
		return td
	}, template.FuncMap{}, "configure_config.html", "wrap.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "wrap.html")
}
