package stremio_list

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/mdblist"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type UserData struct {
	MDBListAPIkey string `json:"mdblist_api_key"`
	MDBListLists  []int  `json:"mdblist_lists"`

	encoded string `json:"-"` // correctly configured

	mdblistListURLs []string `json:"-"`
	mdblistById     map[int]mdblist.MDBListList
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "list",
})

func (ud UserData) HasRequiredValues() bool {
	return ud.MDBListAPIkey != ""
}

func (ud *UserData) GetEncoded() string {
	return ud.encoded
}

func (ud *UserData) SetEncoded(encoded string) {
	ud.encoded = encoded
}

func (ud *UserData) Ptr() *UserData {
	return ud
}

type userDataError struct {
	mdblist struct {
		api_key  string
		list_url []string
	}
}

func (uderr userDataError) HasError() bool {
	if uderr.mdblist.api_key != "" {
		return true
	}
	for i := range uderr.mdblist.list_url {
		if uderr.mdblist.list_url[i] != "" {
			return true
		}
	}
	return false
}

func (uderr userDataError) Error() string {
	var str strings.Builder
	if uderr.mdblist.api_key != "" {
		str.WriteString("mdblist.api_key: " + uderr.mdblist.api_key + "\n")
	}
	for i, err := range uderr.mdblist.list_url {
		if err != "" {
			str.WriteString("mdblist.list[" + strconv.Itoa(i) + "].url: " + err + "\n")
		}
	}
	return str.String()
}

func getUserData(r *http.Request) (*UserData, error) {
	ud := &UserData{
		mdblistById: map[int]mdblist.MDBListList{},
	}
	ud.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(ud); err != nil {
			return nil, err
		}
		if ud.encoded == "" {
			return ud, nil
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		ud.MDBListAPIkey = r.Form.Get("mdblist_api_key")

		lists_length := 0
		if v := r.Form.Get("mdblist_lists_length"); v != "" {
			if lists_length, err = strconv.Atoi(v); err != nil {
				return nil, err
			}
		}

		isMDBListEnabled := ud.MDBListAPIkey != "" || lists_length > 0

		if isMDBListEnabled {
			if ud.MDBListAPIkey == "" {
				err := userDataError{}
				err.mdblist.api_key = "Missing API Key"
				return ud, err
			}

			if lists_length == 0 {
				err := userDataError{}
				err.mdblist.list_url = []string{"Missing List URL"}
				return ud, err
			}

			ud.MDBListLists = make([]int, lists_length)
			ud.mdblistListURLs = make([]string, lists_length)
			udErr := userDataError{}
			udErr.mdblist.list_url = make([]string, lists_length)
			for idx := range lists_length {
				idxStr := strconv.Itoa(idx)
				listUrlStr := r.Form.Get("mdblist_lists[" + idxStr + "].url")
				ud.MDBListLists[idx] = 0
				ud.mdblistListURLs[idx] = listUrlStr

				if listUrlStr == "" {
					udErr.mdblist.list_url[idx] = "Missing List URL"
					continue
				}
				listUrl, err := url.Parse(listUrlStr)
				if err != nil {
					udErr.mdblist.list_url[idx] = "Invalid List URL: " + err.Error()
					continue
				}
				query := listUrl.Query()
				list := mdblist.MDBListList{}
				if idStr := query.Get("list"); idStr != "" {
					id, err := strconv.Atoi(idStr)
					if err != nil {
						udErr.mdblist.list_url[idx] = "Invalid List ID: " + err.Error()
						continue
					}
					list.Id = id
				} else if strings.HasPrefix(listUrl.Path, "/lists/") {
					username, slug, _ := strings.Cut(strings.TrimPrefix(listUrl.Path, "/lists/"), "/")
					if username != "" && slug != "" && !strings.Contains(slug, "/") {
						list.UserName = username
						list.Slug = slug
					} else {
						udErr.mdblist.list_url[idx] = "Invalid List URL"
						continue
					}
				} else {
					udErr.mdblist.list_url[idx] = "Invalid List URL"
					continue
				}

				err = list.Fetch(ud.MDBListAPIkey)
				if err != nil {
					udErr.mdblist.list_url[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.mdblistById[list.Id] = list
				ud.MDBListLists[idx] = list.Id
			}

			if udErr.HasError() {
				return ud, udErr
			}
		}
	}

	return ud, nil
}

func (ud *UserData) FetchListById(id int) (*mdblist.MDBListList, error) {
	if list, ok := ud.mdblistById[id]; ok {
		return &list, nil
	}
	list := mdblist.MDBListList{Id: id}
	err := list.Fetch(ud.MDBListAPIkey)
	ud.mdblistById[list.Id] = list
	return &list, err
}
