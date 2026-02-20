package stremio_store

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

type UserData struct {
	StoreName    string `json:"store_name"`
	StoreToken   string `json:"store_token"`
	HideCatalog  bool   `json:"hide_catalog,omitempty"`
	HideStream   bool   `json:"hide_stream,omitempty"`
	EnableWebDL  bool   `json:"webdl,omitempty"`
	EnableUsenet bool   `json:"usenet,omitempty"`
	encoded      string `json:"-"`

	idPrefixes []string `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	return ud.StoreToken != ""
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

func (ud UserData) StripSecrets() UserData {
	ud.StoreToken = ""
	return ud
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "store",
})

func (ud *UserData) getIdPrefixes() []string {
	if len(ud.idPrefixes) == 0 {
		if ud.StoreName == "" {
			if user, err := util.ParseBasicAuth(ud.StoreToken); err == nil {
				if password := config.Auth.GetPassword(user.Username); password != "" && password == user.Password {
					for _, name := range config.StoreAuthToken.ListStores(user.Username) {
						storeName := store.StoreName(name)
						storeCode := "st-" + string(storeName.Code())
						ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(storeCode))
						if storeName == store.StoreNameTorBox {
							code := storeCode + "-usenet"
							ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(code))

							if ud.EnableWebDL {
								code := storeCode + "-webdl"
								ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(code))
							}
						}
					}
				}
			}
		} else {
			storeName := store.StoreName(ud.StoreName)
			storeCode := string(storeName.Code())
			ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(storeCode))
			if storeName == store.StoreNameTorBox {
				code := storeCode + "-usenet"
				ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(code))

				if ud.EnableWebDL {
					code := storeCode + "-webdl"
					ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(code))
				}
			}
		}
	}
	return ud.idPrefixes
}

type userDataError struct {
	storeToken string
	storeName  string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	if uderr.storeName != "" {
		str.WriteString("store_name: ")
		str.WriteString(uderr.storeName)
		hasSome = true
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.storeToken != "" {
		str.WriteString("store_token: ")
		str.WriteString(uderr.storeToken)
	}
	return str.String()
}

func (ud UserData) GetRequestContext(r *http.Request, idr *ParsedId) (*Ctx, error) {
	ctx := &Ctx{}
	ctx.Log = server.GetReqCtx(r).Log

	storeToken := ud.StoreToken
	if idr.isST {
		user, err := util.ParseBasicAuth(storeToken)
		if err != nil {
			return ctx, &userDataError{storeToken: err.Error()}
		}
		password := config.Auth.GetPassword(user.Username)
		if password != "" && password == user.Password {
			ctx.IsProxyAuthorized = true
			ctx.ProxyAuthUser = user.Username
			ctx.ProxyAuthPassword = user.Password

			if idr.storeName == "" {
				idr.storeName = store.StoreName(config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser))
			}
			storeToken = config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(idr.storeName))
		}
	}

	if idr.storeName == store.StoreNameStremThru {
		ctx.Store = shared.GetStore(string(idr.storeName))
		ctx.StoreAuthToken = ud.StoreToken
	} else if storeToken != "" {
		ctx.Store = shared.GetStore(string(idr.storeName))
		ctx.StoreAuthToken = storeToken
	}

	ctx.ClientIP = shared.GetClientIP(r, &ctx.Context)

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}
	data.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(data); err != nil {
			if errors.Is(err, stremio_userdata.ErrUnsupportedUserdataFormat) {
				return nil, server.ErrorBadRequest(r).WithMessage(err.Error())
			} else {
				return nil, err
			}
		}
		if data.encoded == "" {
			return data, nil
		}
	}

	if IsMethod(r, http.MethodPost) {
		data.StoreName = r.FormValue("store_name")
		data.StoreToken = r.FormValue("store_token")
		data.HideCatalog = r.FormValue("hide_catalog") == "on"
		data.HideStream = r.FormValue("hide_stream") == "on"
		data.EnableWebDL = r.FormValue("enable_webdl") == "on"
		data.EnableUsenet = r.FormValue("enable_usenet") == "on"
	}

	return data, nil
}
