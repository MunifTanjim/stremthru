package stremio_store

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/store"
)

type UserData struct {
	StoreName   string `json:"store_name"`
	StoreToken  string `json:"store_token"`
	HideCatalog bool   `json:"hide_catalog,omitempty"`
	HideStream  bool   `json:"hide_stream,omitempty"`
	EnableWebDL bool   `json:"webdl,omitempty"`
	encoded     string `json:"-"`

	idPrefixes []string `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	return ud.StoreToken != ""
}

func (ud UserData) GetEncoded() (string, error) {
	if ud.encoded != "" {
		return ud.encoded, nil
	}

	blob, err := json.Marshal(ud)
	if err != nil {
		return "", err
	}
	return core.Base64Encode(string(blob)), nil
}

func (ud *UserData) getIdPrefixes() []string {
	if len(ud.idPrefixes) == 0 {
		if ud.StoreName == "" {
			if user, err := core.ParseBasicAuth(ud.StoreToken); err == nil {
				if password := config.ProxyAuthPassword.GetPassword(user.Username); password != "" && password == user.Password {
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

func (ud UserData) GetRequestContext(r *http.Request, idr *ParsedId) (*context.StoreContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &context.StoreContext{
		Log: rCtx.Log,
	}

	storeToken := ud.StoreToken
	if idr.isST {
		user, err := core.ParseBasicAuth(storeToken)
		if err != nil {
			return ctx, &userDataError{storeToken: err.Error()}
		}
		password := config.ProxyAuthPassword.GetPassword(user.Username)
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

	if storeToken != "" {
		ctx.Store = shared.GetStore(string(idr.storeName))
		ctx.StoreAuthToken = storeToken
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		data.encoded = r.PathValue("userData")
		if data.encoded == "" {
			return data, nil
		}
		blob, err := core.Base64DecodeToByte(data.encoded)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(blob, data)
		return data, err
	}

	if IsMethod(r, http.MethodPost) {
		data.StoreName = r.FormValue("store_name")
		data.StoreToken = r.FormValue("store_token")
		data.HideCatalog = r.FormValue("hide_catalog") == "on"
		data.HideStream = r.FormValue("hide_stream") == "on"
		data.EnableWebDL = r.FormValue("enable_webdl") == "on"
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	return data, nil
}
