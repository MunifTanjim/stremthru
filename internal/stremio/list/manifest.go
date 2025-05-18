package stremio_list

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

type parsedId struct {
	Service string
	Id      string
}

func parseId(id string) parsedId {
	id = strings.TrimPrefix(id, "st.list.")
	service, id, _ := strings.Cut(id, ".")
	return parsedId{
		Service: service,
		Id:      id,
	}
}

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
		for _, listId := range ud.MDBListLists {
			list := mdblist.MDBListList{Id: listId}
			err := list.Fetch(ud.MDBListAPIkey)
			if err != nil {
				return nil, err
			}
			catalog := stremio.Catalog{
				Type: string(mediaTypeToResourceType(list.Mediatype)),
				Id:   "st.list.mdblist." + strconv.Itoa(list.Id),
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
			catalogs = append(catalogs, catalog)
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

	SendResponse(w, r, 200, manifest)
}
