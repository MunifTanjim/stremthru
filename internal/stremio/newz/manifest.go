package stremio_newz

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

func GetManifest(r *http.Request, ud *UserData) *stremio.Manifest {
	isConfigured := ud.HasRequiredValues()

	id := shared.GetReversedHostname(r) + ".newz"
	name := "StremThru Newz"
	description := "Stremio Addon for Newz"

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

		for i := range ud.Indexers {
			if i > 0 {
				description += " \n\n"
			}
			hostname := ""
			if iUrl, err := url.Parse(ud.Indexers[i].URL); err == nil {
				hostname = iUrl.Host
			} else {
				hostname, _, _ = strings.Cut(strings.TrimPrefix(strings.TrimPrefix(ud.Indexers[i].URL, "http://"), "https://"), "/")
			}
			description += "(" + hostname + ")"
		}
	}

	streamResource := stremio.Resource{
		Name: stremio.ResourceNameStream,
		Types: []stremio.ContentType{
			stremio.ContentTypeMovie,
			stremio.ContentTypeSeries,
		},
		IDPrefixes: []string{"tt"},
	}

	if config.Feature.IsEnabled(config.FeatureAnime) {
		streamResource.Types = append(streamResource.Types, "anime")
		streamResource.IDPrefixes = append(streamResource.IDPrefixes, "kitsu:", "mal:")
	}

	manifest := &stremio.Manifest{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     config.Version,
		Resources: []stremio.Resource{
			streamResource,
		},
		Types:    []stremio.ContentType{},
		Catalogs: []stremio.Catalog{},
		Logo:     "https://emojiapi.dev/api/v1/newspaper/256.png",
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
