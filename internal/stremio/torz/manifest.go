package stremio_torz

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

func GetManifest(r *http.Request, ud *UserData) *stremio.Manifest {
	isConfigured := ud.HasRequiredValues()

	id := shared.GetReversedHostname(r) + ".torz"
	name := "StremThru Torz"
	description := "Stremio Addon to access crowdsourced Torz"

	if isConfigured {
		storeHint := ""
		for i := range ud.Stores {
			code := string(ud.Stores[i].Code)
			if code == "" {
				code = "st"
			}
			if i > 0 {
				storeHint += " | "
			}
			storeHint += code
		}
		if storeHint != "" {
			storeHint = strings.ToUpper(storeHint)
		}

		description += " — " + storeHint
	}

	manifest := &stremio.Manifest{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     config.Version,
		Resources: []stremio.Resource{
			{
				Name: stremio.ResourceNameStream,
				Types: []stremio.ContentType{
					stremio.ContentTypeMovie,
					stremio.ContentTypeSeries,
				},
				IDPrefixes: []string{"tt"},
			},
		},
		Types:    []stremio.ContentType{},
		Catalogs: []stremio.Catalog{},
		Logo:     "https://emojiapi.dev/api/v1/sparkles/256.png",
		BehaviorHints: &stremio.BehaviorHints{
			Configurable:          true,
			ConfigurationRequired: !isConfigured,
		},
	}

	return manifest
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

	manifest := GetManifest(r, ud)

	SendResponse(w, r, 200, manifest)
}
