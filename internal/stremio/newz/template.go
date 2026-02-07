package stremio_newz

import (
	"bytes"
	"html/template"
	"net/http"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type Base = stremio_template.BaseData

type TemplateDataIndexer struct {
	Type   configure.Config
	Name   configure.Config
	URL    configure.Config
	APIKey configure.Config
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

	Indexers         []TemplateDataIndexer
	CanAddIndexer    bool
	CanRemoveIndexer bool

	Stores           []StoreConfig
	StoreCodeOptions []configure.ConfigOption
	Mode             configure.Config
	CachedOnly       configure.Config

	Error       string
	ManifestURL string
	Script      template.JS

	CanAddStore    bool
	CanRemoveStore bool

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string

	SortConfig   configure.Config
	FilterConfig configure.Config
}

func (td *TemplateData) HasIndexerError() bool {
	for i := range td.Indexers {
		if td.Indexers[i].Type.Error != "" || td.Indexers[i].Name.Error != "" || td.Indexers[i].URL.Error != "" || td.Indexers[i].APIKey.Error != "" {
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
	if td.HasIndexerError() || td.HasStoreError() {
		return true
	}
	return false
}

func getIndexerTypeOptions() []configure.ConfigOption {
	options := []configure.ConfigOption{
		{
			Value: string(stremio_userdata.NewzIndexerTypeGeneric),
			Label: "Generic",
		},
		{
			Value: string(stremio_userdata.NewzIndexerTypeStremThru),
			Label: "StremThru",
		},
		{
			Value: string(stremio_userdata.NewzIndexerTypeTorbox),
			Label: "Torbox",
		},
	}
	return options
}

func newTemplateDataIndexer(index int, indexerType, name, url, apiKey string) TemplateDataIndexer {
	if indexerType == "" {
		indexerType = string(stremio_userdata.NewzIndexerTypeGeneric)
	}
	idx := strconv.Itoa(index)
	isURLDisabled := indexerType == string(stremio_userdata.NewzIndexerTypeTorbox) || indexerType == string(stremio_userdata.NewzIndexerTypeStremThru)
	return TemplateDataIndexer{
		Type: configure.Config{
			Key:      "indexers[" + idx + "].type",
			Type:     configure.ConfigTypeSelect,
			Default:  indexerType,
			Title:    "Type",
			Options:  getIndexerTypeOptions(),
			Required: true,
		},
		Name: configure.Config{
			Key:      "indexers[" + idx + "].name",
			Type:     configure.ConfigTypeText,
			Default:  name,
			Title:    "Name",
			Required: true,
		},
		URL: configure.Config{
			Key:      "indexers[" + idx + "].url",
			Type:     configure.ConfigTypeURL,
			Default:  url,
			Title:    "URL",
			Required: !isURLDisabled,
			Disabled: isURLDisabled,
		},
		APIKey: configure.Config{
			Key:     "indexers[" + idx + "].apikey",
			Type:    configure.ConfigTypePassword,
			Default: apiKey,
			Title:   "API Key",
		},
	}
}

func getModeOptions() []configure.ConfigOption {
	modes := []configure.ConfigOption{}
	modes = append(modes, configure.ConfigOption{
		Label: "Both",
		Value: string(UserDataModeBoth),
	})
	modes = append(modes, configure.ConfigOption{
		Label: "Debrid",
		Value: string(UserDataModeDebrid),
	})
	modes = append(modes, configure.ConfigOption{
		Label: "Stream",
		Value: string(UserDataModeStream),
	})
	return modes
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Newz",
			Description: "Stremio Addon for Newz",
			NavTitle:    "Newz",
		},
		Indexers:         []TemplateDataIndexer{},
		Stores:           []StoreConfig{},
		StoreCodeOptions: stremio_shared.GetStoreCodeOptionsForNewz(),
		Mode: configure.Config{
			Key:     "mode",
			Type:    configure.ConfigTypeSelect,
			Default: configure.ToCheckboxDefault(ud.CachedOnly),
			Title:   "Mode",
			Options: getModeOptions(),
		},
		CachedOnly: configure.Config{
			Key:     "cached",
			Type:    configure.ConfigTypeCheckbox,
			Default: configure.ToCheckboxDefault(ud.CachedOnly),
			Title:   "Only Show Cached Content",
			Tooltip: "Only affects download mode",
		},
		Script: configure.GetScriptStoreTokenDescription("", ""),
		SortConfig: configure.Config{
			Key:         "sort",
			Type:        "text",
			Default:     ud.Sort,
			Title:       "Stream Sort",
			Description: "Comma separated fields: <code>resolution</code>, <code>quality</code>, <code>size</code>, <code>hdr</code>. Prefix with <code>-</code> for reverse sort. Default: <code>" + stremio_transformer.StreamDefaultSortConfig + "</code>",
		},
		FilterConfig: configure.Config{
			Key:         "filter",
			Type:        "textarea",
			Default:     ud.Filter,
			Title:       "Stream Filter",
			Description: `Filter expression, check <a href="https://github.com/MunifTanjim/stremthru/wiki/Stream-Filter" target="_blank">documentation</a>.`,
		},
	}

	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		td.Indexers = append(td.Indexers, newTemplateDataIndexer(
			i,
			string(indexer.Type),
			indexer.Name,
			indexer.URL,
			indexer.APIKey,
		))
	}

	if len(ud.Indexers) == 0 {
		td.Indexers = append(td.Indexers, newTemplateDataIndexer(0, "", "", "", ""))
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
	return stremio_template.GetExecutor("stremio/newz", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.IsTrusted = config.IsTrusted

		td.CanAuthorize = !IsPublicInstance

		td.CanAddIndexer = td.IsAuthed
		td.CanRemoveIndexer = len(td.Indexers) > 1

		td.CanAddStore = td.IsAuthed
		if td.CanAddStore {
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
	}, template.FuncMap{}, "configure_config.html", "newz.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "newz.html")
}
