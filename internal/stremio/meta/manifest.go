package stremio_meta

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var IsPublicInstance = config.IsPublicInstance
var TVDBEnabled = config.Integration.TVDB.IsEnabled()

func GetManifest(r *http.Request, ud *UserData) (*stremio.Manifest, error) {
	id := shared.GetReversedHostname(r) + ".meta"
	name := "StremThru Meta"
	description := "Stremio Addon for Metadata"

	metaResource := stremio.Resource{
		Name:       stremio.ResourceNameMeta,
		Types:      []stremio.ContentType{},
		IDPrefixes: []string{"tt"},
	}

	if ud.Provider.Movie != "" {
		metaResource.Types = append(metaResource.Types, stremio.ContentTypeMovie)
	}
	if ud.Provider.Series != "" {
		metaResource.Types = append(metaResource.Types, stremio.ContentTypeSeries)
	}
	if ud.Provider.Anime != "" {
		metaResource.Types = append(metaResource.Types, "anime")
	}

	if TVDBEnabled {
		metaResource.IDPrefixes = append(metaResource.IDPrefixes, "tvdb:")
	}

	manifest := &stremio.Manifest{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     config.Version,
		Resources:   []stremio.Resource{},
		Types:       []stremio.ContentType{},
		Catalogs:    []stremio.Catalog{},
		Logo:        "https://emojiapi.dev/api/v1/sparkles/256.png",
		BehaviorHints: &stremio.BehaviorHints{
			Configurable:          true,
			ConfigurationRequired: len(metaResource.Types) == 0,
		},
	}

	if len(metaResource.IDPrefixes) > 0 {
		manifest.Resources = append(manifest.Resources, metaResource)
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
