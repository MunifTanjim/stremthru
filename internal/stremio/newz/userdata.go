package stremio_newz

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type UserDataMode string

var (
	UserDataModeBoth   UserDataMode = "debrid+stream"
	UserDataModeDebrid UserDataMode = "debrid"
	UserDataModeStream UserDataMode = "stream"
)

func (m UserDataMode) IsValid() bool {
	switch m {
	case UserDataModeBoth, UserDataModeDebrid, UserDataModeStream:
		return true
	default:
		return false
	}
}

func (m UserDataMode) Contains(mode UserDataMode) bool {
	return slices.Contains(strings.Split(string(m), "+"), string(mode))
}

func (m UserDataMode) Debrid() bool {
	return m == UserDataModeDebrid || m.Contains(UserDataModeDebrid)
}

func (m UserDataMode) Stream() bool {
	return m == UserDataModeStream || m.Contains(UserDataModeStream)
}

type UserData struct {
	stremio_userdata.UserDataNewzIndexers
	stremio_userdata.UserDataStores
	CachedOnly bool         `json:"cached,omitempty"`
	Mode       UserDataMode `json:"mode,omitempty"`
	Sort       string       `json:"sort,omitempty"`
	Filter     string       `json:"filter,omitempty"`

	encoded string `json:"-"` // correctly configured
}

func (ud UserData) StripSecrets() UserData {
	ud.UserDataNewzIndexers = ud.UserDataNewzIndexers.StripSecrets()
	ud.UserDataStores = ud.UserDataStores.StripSecrets()
	return ud
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "newz",
})

func (ud UserData) HasRequiredValues() bool {
	if !ud.UserDataNewzIndexers.HasRequiredValues() {
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
	indexerType   []string
	indexerName   []string
	indexerURL    []string
	indexerAPIKey []string

	storeCode  []string
	storeToken []string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	for i, err := range uderr.indexerType {
		if err == "" {
			continue
		}
		if hasSome {
			str.WriteString(", ")
			hasSome = false
		}
		str.WriteString("indexers[" + strconv.Itoa(i) + "].type: ")
		str.WriteString(err)
		hasSome = true
	}
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

type Indexer struct {
	newznab_client.Indexer
	Id   string
	Name string
}

type Ctx struct {
	stremio_shared.Ctx
	Indexers []Indexer
}

func (ud *UserData) GetRequestContext(r *http.Request) (*Ctx, error) {
	ctx := &Ctx{}
	ctx.Log = server.GetReqCtx(r).Log

	if err, errField := ud.UserDataStores.Prepare(&ctx.Ctx); err != nil {
		switch errField {
		case "store":
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		case "token":
			return ctx, &userDataError{storeToken: []string{err.Error()}}
		default:
			return ctx, &userDataError{storeCode: []string{err.Error()}}
		}
	}

	if !ud.HasStores() {
		if !ud.IsStremThruStore() {
			return ctx, &userDataError{storeCode: []string{"no configured store"}}
		}
		pool, err := usenetmanager.GetPool(log)
		if err != nil {
			return ctx, &userDataError{storeCode: []string{fmt.Errorf("no configured store, failed to check usenet provider: %w", err).Error()}}
		}
		if pool.CountProviders() == 0 {
			return ctx, &userDataError{storeCode: []string{"no configured store or usenet provider"}}
		}
	}

	ctx.ClientIP = shared.GetClientIP(r, &ctx.Context)

	if indexers, err := ud.UserDataNewzIndexers.Prepare(); err != nil {
		return ctx, &userDataError{indexerURL: []string{err.Error()}}
	} else {
		ctx.Indexers = []Indexer{}
		for i, indexer := range indexers {
			ctx.Indexers = append(ctx.Indexers, Indexer{
				Indexer: indexer,
				Id:      indexer.GetId(),
				Name:    ud.UserDataNewzIndexers.Indexers[i].Name,
			})
		}
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

		for i := range util.SafeParseInt(r.Form.Get("indexers_length"), 1) {
			idx := strconv.Itoa(i)
			indexerType := r.Form.Get("indexers[" + idx + "].type")
			name := r.Form.Get("indexers[" + idx + "].name")
			url := r.Form.Get("indexers[" + idx + "].url")
			apiKey := r.Form.Get("indexers[" + idx + "].apikey")

			data.Indexers = append(data.Indexers, stremio_userdata.NewzIndexer{
				Type:   stremio_userdata.NewzIndexerType(indexerType),
				Name:   name,
				URL:    url,
				APIKey: apiKey,
			})
		}

		for idx := range util.SafeParseInt(r.Form.Get("stores_length"), 1) {
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

		data.Mode = UserDataMode(r.Form.Get("mode"))
		if !data.Mode.IsValid() {
			data.Mode = UserDataModeDebrid
		}

		data.CachedOnly = r.Form.Get("cached") == "on"

		data.Sort = r.Form.Get("sort")
		data.Filter = r.Form.Get("filter")
	}

	if IsPublicInstance {
		data.Indexers = data.Indexers[0:0]
		data.Stores = data.Stores[0:0]
	}

	return data, nil
}
