package stremio_list

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/tmdb"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
	"github.com/MunifTanjim/stremthru/stremio"
)

func mdblistMediaTypeToResourceType(mediaType mdblist.MediaType, fallbackMediaType string) stremio.ContentType {
	switch mediaType {
	case mdblist.MediaTypeMovie:
		return stremio.ContentTypeMovie
	case mdblist.MediaTypeShow:
		return stremio.ContentTypeSeries
	default:
		return stremio.ContentType(fallbackMediaType)
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
					Name: list.GetDisplayName(),
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
				list := mdblist.MDBListList{Id: idStr}
				if err := list.Fetch(ud.MDBListAPIkey); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: string(mdblistMediaTypeToResourceType(list.Mediatype, "MDBList")),
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

			case "tmdb":
				list := tmdb.TMDBList{Id: idStr}
				if err := list.Fetch(ud.TMDBTokenId); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: "TMDB",
					Id:   "st.list.tmdb." + idStr,
					Name: list.Name,
					Extra: []stremio.CatalogExtra{
						{
							Name: "skip",
						},
					},
				}

				if list.IsDynamic() {
					meta := tmdb.GetDynamicListMeta(idStr)

					switch meta.MediaType {
					case tmdb.MediaTypeMovie:
						catalog.Type = string(stremio.ContentTypeMovie)
						catalog.Extra = append(catalog.Extra, stremio.CatalogExtra{
							Name:    "genre",
							Options: tmdb.MovieGenres,
						})
					case tmdb.MediaTypeTVShow:
						catalog.Type = string(stremio.ContentTypeSeries)
						catalog.Extra = append(catalog.Extra, stremio.CatalogExtra{
							Name:    "genre",
							Options: tmdb.TVGenres,
						})
					}
				} else {
					catalog.Extra = append(catalog.Extra, stremio.CatalogExtra{
						Name:    "genre",
						Options: tmdb.Genres,
					})
				}
				if hasListNames {
					if name := ud.ListNames[idx]; name != "" {
						catalog.Name = name
					}
				}
				catalogs = append(catalogs, catalog)

			case "trakt":
				list := trakt.TraktList{Id: idStr}
				if err := list.Fetch(ud.TraktTokenId); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: "Trakt",
					Id:   "st.list.trakt." + idStr,
					Name: list.Name,
					Extra: []stremio.CatalogExtra{
						{
							Name:    "genre",
							Options: trakt.Genres,
						},
						{
							Name: "skip",
						},
					},
				}
				if list.IsDynamic() {
					meta := trakt.GetDynamicListMeta(idStr)
					otok, err := ud.getTraktToken()
					if err != nil {
						return nil, err
					}
					if meta.HasUserId && meta.UserId != otok.UserId {
						catalog.Name = meta.UserId + " / " + catalog.Name
					}
					switch meta.ItemType {
					case trakt.ItemTypeMovie:
						catalog.Type = string(stremio.ContentTypeMovie)
					case trakt.ItemTypeShow:
						catalog.Type = string(stremio.ContentTypeSeries)
					}
				}
				if hasListNames {
					if name := ud.ListNames[idx]; name != "" {
						catalog.Name = name
					}
				}
				catalogs = append(catalogs, catalog)

			case "tvdb":
				list := tvdb.TVDBList{Id: idStr}
				if err := list.Fetch(); err != nil {
					return nil, err
				}
				catalog := stremio.Catalog{
					Type: "TVDB",
					Id:   "st.list.tvdb." + idStr,
					Name: list.Name,
					Extra: []stremio.CatalogExtra{
						{
							Name: "skip",
						},
					},
				}

				catalog.Extra = append(catalog.Extra, stremio.CatalogExtra{
					Name:    "genre",
					Options: tvdb.GenreNames,
				})
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
