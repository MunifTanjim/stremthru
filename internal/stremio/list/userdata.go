package stremio_list

import (
	"net/http"

	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type UserData struct {
	MDBListAPIkey string `json:"mdblist_api_key"`

	encoded string `json:"-"` // correctly configured
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

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}
	data.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(data); err != nil {
			return nil, err
		}
		if data.encoded == "" {
			return data, nil
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		data.MDBListAPIkey = r.Form.Get("mdblist_api_key")
	}

	return data, nil
}
