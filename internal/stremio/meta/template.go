package stremio_meta

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

func GetProviderOptions(resourceType string) []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Label: "", Value: ""},
	}
	switch resourceType {
	case "movie", "show", "anime":
		if TVDBEnabled {
			options = append(options, configure.ConfigOption{
				Label: "TVDB",
				Value: "tvdb",
			})
		}
	}
	return options
}

func GetStreamIdProviderOptions(resourceType string) []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Label: "", Value: ""},
		{Label: "IMDB", Value: "imdb"},
	}
	switch resourceType {
	case "movie", "show", "anime":
		if TVDBEnabled {
			options = append(options, configure.ConfigOption{
				Label: "TVDB",
				Value: "tvdb",
			})
		}
	}
	return options
}

type Base = stremio_template.BaseData

type TemplateDataProvider struct {
	Movie  configure.Config
	Series configure.Config
	Anime  configure.Config
}

type TemplateDataStreamId struct {
	Movie  configure.Config
	Series configure.Config
	Anime  configure.Config
}

type TemplateData struct {
	Base

	ManifestURL string
	Script      template.JS

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string

	stremio_userdata.TemplateDataUserData

	Provider TemplateDataProvider
	StreamId TemplateDataStreamId
}

func (td *TemplateData) HasFieldError() bool {
	if td.Provider.Movie.Error != "" ||
		td.Provider.Series.Error != "" ||
		td.Provider.Anime.Error != "" {
		return true
	}
	return false
}

func getTemplateData(ud *UserData, udErr userDataError, isAuthed bool, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru Meta",
			Description: "Stremio Addon for Metadata",
			NavTitle:    "Meta",
		},
		Script: ``,

		Provider: TemplateDataProvider{
			Movie: configure.Config{
				Key:     "provider.movie",
				Title:   "Movie",
				Type:    configure.ConfigTypeSelect,
				Default: ud.Provider.Movie,
				Error:   udErr.provider.movie,
				Options: GetProviderOptions("movie"),
			},
			Series: configure.Config{
				Key:     "provider.series",
				Title:   "Series",
				Type:    configure.ConfigTypeSelect,
				Default: ud.Provider.Series,
				Error:   udErr.provider.series,
				Options: GetProviderOptions("series"),
			},
			Anime: configure.Config{
				Key:     "provider.anime",
				Title:   "Anime",
				Type:    configure.ConfigTypeSelect,
				Default: ud.Provider.Anime,
				Error:   udErr.provider.anime,
				Options: GetProviderOptions("anime"),
			},
		},
		StreamId: TemplateDataStreamId{
			Movie: configure.Config{
				Key:     "stream_id.movie",
				Title:   "Movie",
				Type:    configure.ConfigTypeSelect,
				Default: ud.StreamId.Movie,
				Error:   udErr.stream_id.movie,
				Options: GetStreamIdProviderOptions("movie"),
			},
			Series: configure.Config{
				Key:     "stream_id.series",
				Title:   "Series",
				Type:    configure.ConfigTypeSelect,
				Default: ud.StreamId.Series,
				Error:   udErr.stream_id.series,
				Options: GetStreamIdProviderOptions("series"),
			},
			Anime: configure.Config{
				Key:     "stream_id.anime",
				Title:   "Anime",
				Type:    configure.ConfigTypeSelect,
				Default: ud.StreamId.Anime,
				Error:   udErr.stream_id.anime,
				Options: GetStreamIdProviderOptions("anime"),
			},
		},
	}

	td.IsAuthed = isAuthed

	if udManager.IsSaved(ud) {
		td.SavedUserDataKey = udManager.GetId(ud)
	}
	if td.IsAuthed {
		if options, err := stremio_userdata.GetOptions("meta"); err != nil {
			LogError(r, "failed to list saved userdata options", err)
		} else {
			td.SavedUserDataOptions = options
		}
	} else if td.SavedUserDataKey != "" {
		if sud, err := stremio_userdata.Get[UserData]("meta", td.SavedUserDataKey); err != nil {
			LogError(r, "failed to get saved userdata", err)
		} else {
			td.SavedUserDataOptions = []configure.ConfigOption{{Label: sud.Name, Value: td.SavedUserDataKey}}
		}
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/meta", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.CanAuthorize = !IsPublicInstance

		return td
	}, template.FuncMap{}, "configure_config.html", "configure_submit_button.html", "saved_userdata_field.html", "meta.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "meta.html")
}
