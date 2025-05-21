package stremio_wrap

import (
	"bytes"
	"html/template"
	"net/http"
	"regexp"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Wrap",
			Description: "Stremio Addon to Wrap other Addons with StremThru",
			NavTitle:    "Wrap",
		},
		Upstreams:        []UpstreamAddon{},
		Stores:           []StoreConfig{},
		StoreCodeOptions: stremio_shared.GetStoreCodeOptions(),
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
			Description: "Comma separated fields: <code>resolution</code>, <code>quality</code>, <code>size</code>. Prefix with <code>-</code> for reverse sort. Default: <code>" + stremio_transformer.StreamDefaultSortConfig + "</code>",
		},

		ExtractorIds: []string{},
		TemplateIds:  []string{},
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

	hasExtractor := false

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
				hasExtractor = true
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

	if hasExtractor {
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

	if udManager.IsSaved(ud) {
		td.SavedUserDataKey = udManager.GetId(ud)
	}
	if td.IsAuthed {
		if options, err := stremio_userdata.GetOptions("wrap"); err != nil {
			LogError(r, "failed to list saved userdata options", err)
		} else {
			td.SavedUserDataOptions = options
		}
	} else if td.SavedUserDataKey != "" {
		if sud, err := stremio_userdata.Get[UserData]("wrap", td.SavedUserDataKey); err != nil {
			LogError(r, "failed to get saved userdata", err)
		} else {
			td.SavedUserDataOptions = []configure.ConfigOption{{Label: sud.Name, Value: td.SavedUserDataKey}}
		}
	}

	return td
}

type Base = stremio_template.BaseData

type UpstreamAddon struct {
	URL              string
	IsConfigurable   bool
	Error            string
	ExtractorId      string
	Extractor        stremio_transformer.StreamExtractorBlob
	ExtractorError   string
	NoContentProxy   bool
	ReconfigureStore bool
}

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

	Upstreams []UpstreamAddon

	Stores           []StoreConfig
	StoreCodeOptions []configure.ConfigOption

	Configs     []configure.Config
	Error       string
	ManifestURL string
	Script      template.JS

	CanAddUpstream    bool
	CanRemoveUpstream bool
	CanAddStore       bool
	CanRemoveStore    bool

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string

	ExtractorIds  []string
	TemplateIds   []string
	TemplateId    string
	Template      stremio_transformer.StreamTemplateBlob
	TemplateError stremio_transformer.StreamTemplateBlob
	SortConfig    configure.Config

	stremio_userdata.TemplateDataUserData
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
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
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

		td.IsRedacted = !td.IsAuthed && td.SavedUserDataKey != ""
		if td.IsRedacted {
			redacted := "*******"
			upstreamUrlPattern := regexp.MustCompile("^(.+)://([^/]+)/(?:[^/]+/)?(manifest.json)$")
			for i := range td.Upstreams {
				up := &td.Upstreams[i]
				up.URL = upstreamUrlPattern.ReplaceAllString(up.URL, "${1}://${2}/"+redacted+"/${3}")
			}
			for i := range td.Stores {
				s := &td.Stores[i]
				s.Token = redacted
			}
		}

		return td
	}, template.FuncMap{}, "configure_config.html", "configure_submit_button.html", "saved_userdata_field.html", "wrap.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "wrap.html")
}
