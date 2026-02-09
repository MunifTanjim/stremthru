package stremio_store

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var AnimeEnabled = config.Feature.IsEnabled("anime")

const ContentTypeOther = "other"

const (
	CatalogGenreVideo     = "Video"
	CatalogGenreStremThru = "StremThru"
)

var logoByStoreCode = map[string]string{
	"*":  "https://emojiapi.dev/api/v1/sparkles/256.png",
	"ad": "https://cdn.alldebrid.com/lib/images/default/logo_alldebrid.png",
	"dl": "https://debrid-link.com/img/fav/icon_192.png",
	"ed": "https://paradise-cloud.com/android-chrome-192x192.png",
	"oc": "https://offcloud.com/images/apple-touch-icon-180x180.png",
	"pm": "https://www.premiumize.me/apple-touch-icon.png",
	"pp": "https://mypikpak.com/android-chrome-192x192.png",
	"qb": "https://upload.wikimedia.org/wikipedia/commons/6/66/New_qBittorrent_Logo.svg",
	"rd": "https://fcdn.real-debrid.com/0830/favicons/android-chrome-192x192.png",
	"tb": "https://torbox.app/android-chrome-192x192.png",
}

func getManifestCatalog(code string, hideCatalog bool) stremio.Catalog {
	return stremio.Catalog{
		Id:   getCatalogId(code),
		Name: "Store " + strings.ToUpper(code),
		Type: ContentTypeOther,
		Extra: []stremio.CatalogExtra{
			{
				Name:       "search",
				IsRequired: hideCatalog,
			},
			{
				Name: "skip",
			},
			{
				Name:    "genre",
				Options: []string{CatalogGenreVideo, CatalogGenreStremThru},
			},
		},
	}
}

func GetManifest(r *http.Request, ud *UserData) (*stremio.Manifest, error) {
	isConfigured := ud.HasRequiredValues()

	id := shared.GetReversedHostname(r) + ".store"
	name := "Store"
	description := "Explore and Search Store Catalog"
	logo := logoByStoreCode["*"]
	idPrefixes := []string{}
	catalogs := []stremio.Catalog{}
	if isConfigured {
		switch ud.StoreName {
		case "":
			names := []string{}
			if user, err := util.ParseBasicAuth(ud.StoreToken); err == nil {
				if password := config.Auth.GetPassword(user.Username); password != "" && password == user.Password {
					if ud.EnableUsenet {
						storeName := store.StoreNameStremThru
						storeCode := storeName.Code()

						usenetCode := string(storeCode) + "-usenet"
						idPrefixes = append(idPrefixes, getIdPrefix(usenetCode))
						catalogs = append(catalogs, getManifestCatalog(usenetCode, ud.HideCatalog))
					}

					for _, name := range config.StoreAuthToken.ListStores(user.Username) {
						storeName := store.StoreName(name)
						storeCode := storeName.Code()

						s := shared.GetStoreByCode(string(storeCode))
						if s == nil {
							return nil, core.NewError("invalid store code: " + string(storeCode))
						}
						storeToken := config.StoreAuthToken.GetToken(user.Username, string(storeName))
						getUserParams := &store.GetUserParams{}
						getUserParams.APIKey = storeToken
						user, err := s.GetUser(getUserParams)
						if err != nil {
							return nil, err
						}

						names = append(names, string(storeName))

						code := "st-" + string(storeCode)
						idPrefixes = append(idPrefixes, getIdPrefix(code))
						catalogs = append(catalogs, getManifestCatalog(code, ud.HideCatalog))

						if ud.EnableUsenet && stremio_store_usenet.IsSupported(storeCode) && user.HasUsenet {
							usenetCode := code + "-usenet"
							idPrefixes = append(idPrefixes, getIdPrefix(usenetCode))
							catalogs = append(catalogs, getManifestCatalog(usenetCode, ud.HideCatalog))
						}

						if storeName == store.StoreNameTorBox {
							if ud.EnableWebDL {
								webdlCode := code + "-webdl"
								idPrefixes = append(idPrefixes, getIdPrefix(webdlCode))
								catalogs = append(catalogs, getManifestCatalog(webdlCode, ud.HideCatalog))
							}
						}
					}
				}
			}

			id += ".st"
			name = name + " | " + "ST"
			description = description + " - StremThru ( " + strings.Join(names, " | ") + " )"
		default:
			storeName := store.StoreName(ud.StoreName)
			storeCode := string(storeName.Code())

			s := shared.GetStoreByCode(string(storeCode))
			if s == nil {
				return nil, core.NewError("invalid store code: " + string(storeCode))
			}
			storeToken := ud.StoreToken
			getUserParams := &store.GetUserParams{}
			getUserParams.APIKey = storeToken
			user, err := s.GetUser(getUserParams)
			if err != nil {
				return nil, err
			}

			id += "." + storeCode
			name = name + " | " + strings.ToUpper(storeCode)
			description = description + " - " + string(storeName)
			if storeLogo, ok := logoByStoreCode[storeCode]; ok {
				logo = storeLogo
			}

			idPrefixes = append(idPrefixes, getIdPrefix(storeCode))
			catalogs = append(catalogs, getManifestCatalog(storeCode, ud.HideCatalog))

			if ud.EnableUsenet && stremio_store_usenet.IsSupported(storeName.Code()) && user.HasUsenet {
				usenetCode := storeCode + "-usenet"
				idPrefixes = append(idPrefixes, getIdPrefix(usenetCode))
				catalogs = append(catalogs, getManifestCatalog(usenetCode, ud.HideCatalog))
			}

			if storeName == store.StoreNameTorBox {
				if ud.EnableWebDL {
					webdlCode := storeCode + "-webdl"
					idPrefixes = append(idPrefixes, getIdPrefix(webdlCode))
					catalogs = append(catalogs, getManifestCatalog(webdlCode, ud.HideCatalog))
				}
			}
		}
	} else {
		name = "StremThru Store"
	}

	streamResource := stremio.Resource{
		Name:       stremio.ResourceNameStream,
		Types:      []stremio.ContentType{ContentTypeOther},
		IDPrefixes: idPrefixes,
	}
	if !ud.HideStream {
		streamResource.IDPrefixes = append([]string{"tt"}, idPrefixes...)
		streamResource.Types = append(streamResource.Types, stremio.ContentTypeMovie, stremio.ContentTypeSeries)
		if AnimeEnabled {
			streamResource.IDPrefixes = append(streamResource.IDPrefixes, "kitsu:", "mal:")
			streamResource.Types = append(streamResource.Types, "anime")
		}
	}

	manifest := &stremio.Manifest{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     config.Version,
		Logo:        logo,
		Resources: []stremio.Resource{
			{
				Name:       stremio.ResourceNameMeta,
				Types:      []stremio.ContentType{ContentTypeOther},
				IDPrefixes: idPrefixes,
			},
			streamResource,
		},
		Types:    []stremio.ContentType{},
		Catalogs: catalogs,
		BehaviorHints: &stremio.BehaviorHints{
			Configurable:          true,
			ConfigurationRequired: !isConfigured,
		},
	}

	return manifest, nil
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	manifest, err := GetManifest(r, ud)
	if err != nil {
		SendError(w, r, err)
		return
	}

	stremio_shared.ClaimAddonOnStremioAddonsDotNet(manifest, "eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4Q0JDLUhTMjU2In0..9hY0hRGJ_MmMb5g39-Y_pg.shmFmHHoqoxyAv42csODUejG7r65_pLIfv6LxFNakQ1ed_OcyTDU3He79vWCYk42__uCclJZ4ZbpqZO-Oo2khmndQlwQmgTfpwGzPBxqz1oOq0GOn2R3KDyzHln6Lie0.se53UR-KR8hCCZgO4b3daA")

	SendResponse(w, r, 200, manifest)
}
