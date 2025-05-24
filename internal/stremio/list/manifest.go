package stremio_list

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

func mediaTypeToResourceType(mediaType mdblist.MediaType) stremio.ContentType {
	switch mediaType {
	case mdblist.MediaTypeMovie:
		return stremio.ContentTypeMovie
	case mdblist.MediaTypeShow:
		return stremio.ContentTypeSeries
	default:
		return "other"
	}
}

func GetManifest(r *http.Request, ud *UserData) (*stremio.Manifest, error) {
	isConfigured := ud.HasRequiredValues()

	id := shared.GetReversedHostname(r) + ".list"
	name := "StremThru List"
	description := "Stremio Addon for accessing Lists"

	catalogs := []stremio.Catalog{}

	if isConfigured {
		hasListNames := len(ud.ListNames) > 0

		for idx, listId := range ud.Lists {
			service, idStr, ok := strings.Cut(listId, ":")
			if !ok {
				return nil, core.NewError("invalid list id: " + listId)
			}
			switch service {
			case "anilist":
				list := anilist.AniListList{Id: idStr}
				if err := list.Fetch(); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: "anime",
					Id:   "st.list.anilist." + idStr,
					Name: list.GetUserName() + "/" + list.GetName(),
					Extra: []stremio.CatalogExtra{
						{
							Name:    "genre",
							Options: anilist.Genres,
						},
						{
							Name: "skip",
						},
					},
				}
				if hasListNames {
					if name := ud.ListNames[idx]; name != "" {
						catalog.Name = name
					}
				}
				catalogs = append(catalogs, catalog)

			case "mdblist":
				id, err := strconv.Atoi(idStr)
				if err != nil {
					return nil, core.NewError("invalid list id: " + listId)
				}
				list := mdblist.MDBListList{Id: id}
				if err := list.Fetch(ud.MDBListAPIkey); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: string(mediaTypeToResourceType(list.Mediatype)),
					Id:   "st.list.mdblist." + idStr,
					Name: list.Name,
					Extra: []stremio.CatalogExtra{
						{
							Name:    "genre",
							Options: mdblist.Genres,
						},
						{
							Name: "skip",
						},
					},
				}
				if hasListNames {
					if name := ud.ListNames[idx]; name != "" {
						catalog.Name = name
					}
				}
				catalogs = append(catalogs, catalog)
			}
		}
	}

	manifest := &stremio.Manifest{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     config.Version,
		Resources: []stremio.Resource{
			{
				Name: stremio.ResourceNameCatalog,
				Types: []stremio.ContentType{
					stremio.ContentTypeMovie,
					stremio.ContentTypeSeries,
				},
			},
		},
		Types:    []stremio.ContentType{},
		Catalogs: catalogs,
		Logo:     "https://emojiapi.dev/api/v1/sparkles/256.png",
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

	ud, err := getUserData(r, false)
	if err != nil {
		SendError(w, r, err)
		return
	}

	manifest, err := GetManifest(r, ud)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendResponse(w, r, 200, manifest)
}
