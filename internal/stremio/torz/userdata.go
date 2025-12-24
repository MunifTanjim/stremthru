package stremio_torz

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type UserData struct {
	stremio_userdata.UserDataIndexers
	IncludeUncachedPrivate bool `json:"unc_prvt,omitempty"`
	stremio_userdata.UserDataStores
	CachedOnly bool   `json:"cached,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Filter     string `json:"filter,omitempty"`

	encoded string `json:"-"` // correctly configured
}

func (ud UserData) StripSecrets() UserData {
	ud.UserDataIndexers = ud.UserDataIndexers.StripSecrets()
	ud.UserDataStores = ud.UserDataStores.StripSecrets()
	return ud
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "torz",
})

func (ud UserData) HasRequiredValues() bool {
	if !ud.UserDataIndexers.HasRequiredValues() {
		return false
	}
	if !ud.UserDataStores.HasRequiredValues() {
		return false
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
	indexerName   []string
	indexerURL    []string
	indexerAPIKey []string

	storeCode  []string
	storeToken []string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	for i, err := range uderr.indexerName {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("indexers[" + strconv.Itoa(i) + "].name: ")
		str.WriteString(err)
		hasSome = true
	}
	for i, err := range uderr.indexerURL {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("indexers[" + strconv.Itoa(i) + "].url: ")
		str.WriteString(err)
		hasSome = true
	}
	for i, err := range uderr.indexerAPIKey {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("indexers[" + strconv.Itoa(i) + "].apikey: ")
		str.WriteString(err)
		hasSome = true
	}
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

type RequestContext struct {
	*context.StoreContext
	Indexers []torznab_client.Indexer
}

func (ud *UserData) GetRequestContext(r *http.Request) (*RequestContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &RequestContext{
		StoreContext: &context.StoreContext{
			Log: rCtx.Log,
		},
	}

	if err, errField := ud.UserDataStores.Prepare(ctx.StoreContext); err != nil {
		switch errField {
		case "store":
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		case "token":
			return ctx, &userDataError{storeToken: []string{err.Error()}}
		default:
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		}
	}

	if !ud.HasStores() && !ud.IsP2P() {
		return ctx, &userDataError{storeCode: []string{"no configured store"}}
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx.StoreContext)

	if indexers, err := ud.UserDataIndexers.Prepare(); err != nil {
		return ctx, &userDataError{indexerURL: []string{err.Error()}}
	} else {
		ctx.Indexers = indexers
	}

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

		for i := range util.SafeParseInt(r.Form.Get("indexers_length"), 1) {
			idx := strconv.Itoa(i)
			name := r.Form.Get("indexers[" + idx + "].name")
			url := r.Form.Get("indexers[" + idx + "].url")
			apiKey := r.Form.Get("indexers[" + idx + "].apikey")

			data.Indexers = append(data.Indexers, stremio_userdata.Indexer{
				Name:   stremio_userdata.IndexerName(name),
				URL:    url,
				APIKey: apiKey,
			})
		}

		data.Sort = r.Form.Get("sort")
		data.Filter = r.Form.Get("filter")
		data.IncludeUncachedPrivate = r.Form.Get("uncached_private") == "on"
	}

	if IsPublicInstance && len(data.Stores) > MaxPublicInstanceStoreCount {
		data.Stores = data.Stores[0:MaxPublicInstanceStoreCount]
	}

	return data, nil
}
