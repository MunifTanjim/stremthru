package stremio_torz

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
	UncachedPrivate  configure.Config

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

	SortConfig   configure.Config
	FilterConfig configure.Config
}

func (td *TemplateData) HasIndexerError() bool {
	for i := range td.Indexers {
		if td.Indexers[i].Name.Error != "" || td.Indexers[i].URL.Error != "" || td.Indexers[i].APIKey.Error != "" {
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
	for i := range td.Configs {
		if td.Configs[i].Error != "" {
			return true
		}
	}
	return false
}

func getIndexerNameOptions() []configure.ConfigOption {
	options := []configure.ConfigOption{
		// {
		// 	Disabled: true,
		// 	Value:    string(stremio_userdata.IndexerNameGeneric),
		// 	Label:    "Generic",
		// },
		{
			Value: string(stremio_userdata.IndexerNameJackett),
			Label: "Jackett",
		},
	}
	return options
}

func newTemplateDataIndexer(index int, name, url, apiKey string) TemplateDataIndexer {
	return TemplateDataIndexer{
		Name: configure.Config{
			Key:      "indexers[" + strconv.Itoa(index) + "].name",
			Type:     configure.ConfigTypeSelect,
			Default:  name,
			Title:    "Name",
			Options:  getIndexerNameOptions(),
			Required: true,
		},
		URL: configure.Config{
			Key:      "indexers[" + strconv.Itoa(index) + "].url",
			Type:     configure.ConfigTypeURL,
			Default:  url,
			Title:    "URL",
			Required: true,
		},
		APIKey: configure.Config{
			Key:     "indexers[" + strconv.Itoa(index) + "].apikey",
			Type:    configure.ConfigTypePassword,
			Default: apiKey,
			Title:   "API Key",
		},
	}
}

func getTemplateData(ud *UserData, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Torz",
			Description: "Stremio Addon to access crowdsourced Torz",
			NavTitle:    "Torz",
		},
		Indexers: []TemplateDataIndexer{},
		UncachedPrivate: configure.Config{
			Key:     "uncached_private",
			Type:    configure.ConfigTypeCheckbox,
			Default: configure.ToCheckboxDefault(ud.IncludeUncachedPrivate),
			Title:   "Show Uncached Private (only for TorBox)",
		},
		Stores:           []StoreConfig{},
		StoreCodeOptions: stremio_shared.GetStoreCodeOptions(true),
		Configs: []configure.Config{
			{
				Key:   "cached",
				Type:  configure.ConfigTypeCheckbox,
				Title: "Only Show Cached Content",
			},
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
			Title:       "🧪 Stream Filter",
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
			string(indexer.Name),
			indexer.URL,
			indexer.APIKey,
		))
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
		td.IsTrusted = config.IsTrusted

		td.CanAuthorize = !IsPublicInstance

		td.CanAddIndexer = td.IsAuthed || len(td.Indexers) < MaxPublicInstanceIndexerCount
		td.CanRemoveIndexer = len(td.Indexers) > 0

		td.CanAddStore = td.IsAuthed || len(td.Stores) < MaxPublicInstanceStoreCount
		if !IsPublicInstance && td.CanAddStore {
			for i := range td.Stores {
				s := &td.Stores[i]
				if s.Code.IsP2P() || (s.Code.IsStremThru() && s.Token != "") {
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
