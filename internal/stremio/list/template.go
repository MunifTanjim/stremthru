package stremio_list

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_template "github.com/MunifTanjim/stremthru/internal/stremio/template"
)

var IsPublicInstance = config.IsPublicInstance
var MaxPublicInstanceMDBListListCount = 5

type Base = stremio_template.BaseData

type TemplateDataMDBListList struct {
	URL   string
	Error struct {
		URL string
	}
}

type TemplateDataMDBList struct {
	APIKey string
	Error  struct {
		APIKey string
	}
	Lists []TemplateDataMDBListList

	CanAddList    bool
	CanRemoveList bool
}

type TemplateData struct {
	Base

	MDBList TemplateDataMDBList

	ManifestURL string
	Script      template.JS

	CanAuthorize bool
	IsAuthed     bool
	AuthError    string
}

func (td *TemplateData) HasMDBListError() bool {
	if len(td.MDBList.Lists) > 0 {
		if td.MDBList.Error.APIKey != "" {
			return true
		}

		for i := range td.MDBList.Lists {
			if td.MDBList.Lists[i].Error.URL != "" {
				return true
			}
		}

	}
	return false
}

func (td *TemplateData) HasFieldError() bool {
	if td.HasMDBListError() {
		return true
	}
	return false
}

func getTemplateData(ud *UserData, udError userDataError, w http.ResponseWriter, r *http.Request) *TemplateData {
	td := &TemplateData{
		Base: Base{
			Title:       "StremThru List",
			Description: "Stremio Addon to access various Lists",
			NavTitle:    "List",
		},
		MDBList: TemplateDataMDBList{
			APIKey: ud.MDBListAPIkey,
			Lists:  []TemplateDataMDBListList{},
		},
	}

	td.MDBList.Error.APIKey = udError.mdblist.api_key

	if len(ud.MDBListLists) > 0 {
		if ud.MDBListAPIkey != "" && len(ud.mdblistListURLs) == 0 {
			for _, listId := range ud.MDBListLists {
				list := mdblist.MDBListList{Id: listId}
				if err := list.Fetch(ud.MDBListAPIkey); err != nil {
					log.Error("failed to fetch list", "error", err, "id", listId)
					break
				}
				ud.mdblistListURLs = append(ud.mdblistListURLs, list.GetURL())
			}
		}

		for i := range ud.MDBListLists {
			list := TemplateDataMDBListList{}
			list.URL = ud.mdblistListURLs[i]
			if len(udError.mdblist.list_url) > i {
				list.Error.URL = udError.mdblist.list_url[i]
			}
			if list.URL == "" && list.Error.URL == "" {
				list.Error.URL = "Missing List URL"
			}
			td.MDBList.Lists = append(td.MDBList.Lists, list)
		}
	} else {
		td.MDBList.Lists = append(td.MDBList.Lists, TemplateDataMDBListList{})
	}

	if cookie, err := stremio_shared.GetAdminCookieValue(w, r); err == nil && !cookie.IsExpired {
		td.IsAuthed = config.ProxyAuthPassword.GetPassword(cookie.User()) == cookie.Pass()
	}

	return td
}

var executeTemplate = func() stremio_template.Executor[TemplateData] {
	return stremio_template.GetExecutor("stremio/list", func(td *TemplateData) *TemplateData {
		td.StremThruAddons = stremio_shared.GetStremThruAddons()
		td.Version = config.Version
		td.CanAuthorize = !IsPublicInstance
		td.MDBList.CanAddList = td.IsAuthed || len(td.MDBList.Lists) < MaxPublicInstanceMDBListListCount
		td.MDBList.CanRemoveList = len(td.MDBList.Lists) > 1

		return td
	}, template.FuncMap{}, "configure_config.html", "list.html")
}()

func getPage(td *TemplateData) (bytes.Buffer, error) {
	return executeTemplate(td, "list.html")
}
