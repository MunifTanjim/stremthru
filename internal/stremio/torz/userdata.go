package stremio_torz

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type UserDataStoreCode string

type UserDataStore struct {
	Code  UserDataStoreCode `json:"c"`
	Token string            `json:"t"`
}

type UserData struct {
	stremio_userdata.UserDataStores
	CachedOnly bool `json:"cached,omitempty"`

	encoded string `json:"-"` // correctly configured
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "torz",
})

func (ud UserData) HasRequiredValues() bool {
	if len(ud.Stores) == 0 {
		return false
	}
	for i := range ud.Stores {
		s := &ud.Stores[i]
		if s.Code.IsStremThru() && len(ud.Stores) > 1 {
			return false
		}
		if s.Token == "" {
			return false
		}
	}
	return true
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
	storeCode  []string
	storeToken []string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	for i, err := range uderr.storeCode {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("stores[" + strconv.Itoa(i) + "].code: ")
		str.WriteString(err)
		hasSome = true
	}
	for i, err := range uderr.storeToken {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("stores[" + strconv.Itoa(i) + "].token: ")
		str.WriteString(err)
		hasSome = true
	}
	return str.String()
}

func (ud *UserData) GetRequestContext(r *http.Request) (*context.StoreContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &context.StoreContext{
		Log: rCtx.Log,
	}

	if err, errField := ud.UserDataStores.Prepare(ctx); err != nil {
		switch errField {
		case "store":
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		case "token":
			return ctx, &userDataError{storeToken: []string{err.Error()}}
		default:
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		}
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
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

		stores_length := 1
		if v := r.Form.Get("stores_length"); v != "" {
			stores_length, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		}

		for idx := range stores_length {
			code := r.Form.Get("stores[" + strconv.Itoa(idx) + "].code")
			token := r.Form.Get("stores[" + strconv.Itoa(idx) + "].token")
			if code == "" {
				data.Stores = []stremio_userdata.Store{
					{
						Code:  stremio_userdata.StoreCode(code),
						Token: token,
					},
				}
				break
			} else {
				data.Stores = append(data.Stores, stremio_userdata.Store{
					Code:  stremio_userdata.StoreCode(code),
					Token: token,
				})
			}
		}

		data.CachedOnly = r.Form.Get("cached") == "on"
	}

	if IsPublicInstance && len(data.Stores) > MaxPublicInstanceStoreCount {
		data.Stores = data.Stores[0:MaxPublicInstanceStoreCount]
	}

	return data, nil
}
