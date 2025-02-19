package stremio_wrap

import (
	"bytes"
	"html/template"
	"net/http"

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

func getStoreCodeOptions() []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "ad", Label: "AllDebrid"},
		{Value: "dl", Label: "DebridLink"},
		{Value: "ed", Label: "EasyDebrid"},
		{Value: "oc", Label: "Offcloud"},
		{Value: "pm", Label: "Premiumize"},
		{Value: "pp", Label: "PikPak"},
		{Value: "rd", Label: "RealDebrid"},
		{Value: "tb", Label: "TorBox"},
	}
	if !config.ProxyStreamEnabled {
		options[0].Disabled = true
		options[0].Label = ""
	}
	return options
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Wrap",
			Description: "Stremio Addon to Wrap other Addons with StremThru",
			NavTitle:    "Wrap",
		},
		Upstreams:        []UpstreamAddon{},
		Stores:           []StoreConfig{},
		StoreCodeOptions: getStoreCodeOptions(),
		Configs: []configure.Config{
			{
				Key:     "cached",
				Type:    configure.ConfigTypeCheckbox,
				Title:   "Only Show Cached Content",
				Options: []configure.ConfigOption{},
			},
		},
		Script: configure.GetScriptStoreTokenDescription("", ""),

		SortConfig: configure.Config{
			Key:         "sort",
			Type:        "text",
			Default:     ud.Sort,
			Title:       "Stream Sort",
			Description: "Comma separated fields: <code>resolution</code>, <code>quality</code>, <code>size</code>. Prefix with <code>-</code> for reverse sort.",
		},

		ExtractorIds: []string{},
		TemplateIds:  []string{},
	}

	if cookie, err := getCookieValue(w, r); err == nil && !cookie.IsExpired {
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

	isExecutingAction := r.Header.Get("x-addon-configure-action") != ""

	td.TemplateId = ud.TemplateId
	td.Template = ud.template
	if !isExecutingAction {
		if td.TemplateId != "" {
			if storedBlob, err := getTemplate(td.TemplateId); err == nil {
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
				if storedBlob, err := getExtractor(up.ExtractorId); err == nil {
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
			URL:              up.URL,
			ExtractorId:      up.ExtractorId,
			Extractor:        up.extractor,
			ExtractorError:   extractorError,
			NoContentProxy:   up.NoContentProxy,
			ReconfigureStore: up.ReconfigureStore,
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

	if extractorIds, err := getExtractorIds(); err != nil {
		LogError(r, "failed to list extractors", err)
	} else {
		td.ExtractorIds = extractorIds
	}

	if templateIds, err := getTemplateIds(); err != nil {
		LogError(r, "failed to list templates", err)
	} else {
		td.TemplateIds = templateIds
	}

	return td
}

type Base = stremio_template.BaseData

type UpstreamAddon struct {
	URL              string
	IsConfigurable   bool
	Error            string
	ExtractorId      string
	Extractor        StreamTransformerExtractorBlob
	ExtractorError   string
	NoContentProxy   bool
	ReconfigureStore bool
}

type StoreConfig struct {
	Code  UserDataStoreCode
	Token string
	Error struct {
		Code  string
		Token string
	}
}

type TemplateData struct {
	Base

	Upstreams []UpstreamAddon

	Stores           []StoreConfig
	StoreCodeOptions []configure.ConfigOption

	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS

	CanAuthorize      bool
	CanAddUpstream    bool
	CanRemoveUpstream bool
	CanAddStore       bool
	CanRemoveStore    bool

	IsAuthed      bool
	ExtractorIds  []string
	TemplateIds   []string
	TemplateId    string
	Template      StreamTransformerTemplateBlob
	TemplateError StreamTransformerTemplateBlob
	SortConfig    configure.Config
}

func (td *TemplateData) HasUpstreamError() bool {
	for i := range td.Upstreams {
		if td.Upstreams[i].Error != "" || td.Upstreams[i].ExtractorError != "" {
			return true
		}
	}
	return false
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
	if td.HasUpstreamError() {
		return true
	}
	if td.HasStoreError() {
		return true
	}
	if !td.TemplateError.IsEmpty() {
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
		td.CanAuthorize = !IsPublicInstance
		td.CanAddUpstream = td.IsAuthed || len(td.Upstreams) < MaxPublicInstanceUpstreamCount
		td.CanRemoveUpstream = len(td.Upstreams) > 1
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
	}, template.FuncMap{}, "configure_config.html", "wrap.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "wrap.html")
}
