package stremio_list

import (
	"bytes"
	"html/template"
	"net/http"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

var IsPublicInstance = config.IsPublicInstance
var MaxPublicInstanceListCount = 5

type Base = stremio_template.BaseData

type TemplateDataList struct {
	URL   string
	Name  string
	Error struct {
		URL  string
		Name string
	}
}

type TemplateData struct {
	Base

	Lists         []TemplateDataList
	CanAddList    bool
	CanRemoveList bool

	MDBListAPIKey configure.Config

	RPDBAPIKey configure.Config

	Shuffle configure.Config

	ManifestURL string
	Script      template.JS

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string

	stremio_userdata.TemplateDataUserData
}

func (td *TemplateData) HasListError() bool {
	if len(td.Lists) > 0 {
		if td.MDBListAPIKey.Error != "" {
			return true
		}

		for i := range td.Lists {
			if td.Lists[i].Error.URL != "" {
				return true
			}
		}
	}
	return false
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasListError() {
		return true
	}
	return false
}

func getTemplateData(ud *UserData, udError userDataError, isAuthed bool, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru List",
			Description: "Stremio Addon to access various Lists",
			NavTitle:    "List",
		},
		Lists: []TemplateDataList{},
		MDBListAPIKey: configure.Config{
			Key:          "mdblist_api_key",
			Type:         "password",
			Default:      ud.MDBListAPIkey,
			Title:        "MDBList API Key",
			Description:  `<a href="https://mdblist.com/preferences/#api_key_uid" target="_blank">API Key</a>`,
			Autocomplete: "off",
			Error:        udError.mdblist.api_key,
		},
		RPDBAPIKey: configure.Config{
			Key:          "rpdb_api_key",
			Type:         configure.ConfigTypePassword,
			Default:      ud.RPDBAPIKey,
			Title:        "RPDB API Key",
			Description:  `Rating Poster Database <a href="https://ratingposterdb.com/api-key/" target="blank">API Key</a>`,
			Autocomplete: "off",
		},
		Shuffle: configure.Config{
			Key:   "shuffle",
			Type:  configure.ConfigTypeCheckbox,
			Title: "Shuffle List Items",
		},
	}

	if ud.Shuffle {
		td.Shuffle.Default = "checked"
	}

	hasListNames := len(ud.ListNames) > 0
	for i, listId := range ud.Lists {
		list := TemplateDataList{}
		if hasListNames {
			list.Name = ud.ListNames[i]
		}
		if len(ud.list_urls) > i {
			list.URL = ud.list_urls[i]
		}
		if len(udError.list_urls) > i {
			list.Error.URL = udError.list_urls[i]
		}

		if listId == "" {
			if list.Error.URL == "" {
				list.Error.URL = "Missing List ID"
			}
		} else if list.URL == "" {
			service, id, err := parseListId(listId)
			if err != nil {
				list.Error.URL = "Failed to Parse List ID: " + listId
			} else {
				switch service {
				case "mdblist":
					lId, err := strconv.Atoi(id)
					if err != nil {
						list.Error.URL = "Failed to Parse List ID: " + id
					}
					l := mdblist.MDBListList{Id: lId}
					if err := ud.FetchMDBListList(&l); err != nil {
						log.Error("failed to fetch list", "error", err, "id", listId)
						list.Error.URL = "Failed to Fetch List: " + err.Error()
					} else {
						list.URL = l.GetURL()
					}
				}
			}
		}
		if list.URL == "" && list.Error.URL == "" {
			list.Error.URL = "Missing List URL"
		}
		td.Lists = append(td.Lists, list)
	}

	td.IsAuthed = isAuthed

	if udManager.IsSaved(ud) {
		td.SavedUserDataKey = udManager.GetId(ud)
	}
	if td.IsAuthed {
		if options, err := stremio_userdata.GetOptions("list"); err != nil {
			LogError(r, "failed to list saved userdata options", err)
		} else {
			td.SavedUserDataOptions = options
		}
	} else if td.SavedUserDataKey != "" {
		if sud, err := stremio_userdata.Get[UserData]("list", td.SavedUserDataKey); err != nil {
			LogError(r, "failed to get saved userdata", err)
		} else {
			td.SavedUserDataOptions = []configure.ConfigOption{{Label: sud.Name, Value: td.SavedUserDataKey}}
		}
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/list", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.CanAuthorize = !IsPublicInstance
		td.CanAddList = td.IsAuthed || len(td.Lists) < MaxPublicInstanceListCount
		td.CanRemoveList = len(td.Lists) > 1

		if len(td.Lists) == 0 {
			td.Lists = append(td.Lists, TemplateDataList{})
		}

		td.IsRedacted = !td.IsAuthed && td.SavedUserDataKey != ""
		if td.IsRedacted {
			redacted := "*******"
			if td.MDBListAPIKey.Default != "" {
				td.MDBListAPIKey.Default = redacted
			}
			if td.RPDBAPIKey.Default != "" {
				td.RPDBAPIKey.Default = redacted
			}
		}

		return td
	}, template.FuncMap{}, "configure_config.html", "configure_submit_button.html", "saved_userdata_field.html", "list.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "list.html")
}
