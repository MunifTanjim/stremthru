package stremio_disabled

import (
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/stremio"
)

var client = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

func GetDisabledManifest(manifestUrl string) (*stremio.Manifest, error) {
	baseUrl, err := url.Parse(strings.TrimSuffix(manifestUrl, "/manifest.json"))
	if err != nil {
		return nil, err
	}

	res, err := client.GetManifest(&stremio_addon.GetManifestParams{BaseURL: baseUrl})
	if err != nil {
		return nil, err
	}

	manifest := &stremio.Manifest{
		ID:           "st:disabled:" + res.Data.ID,
		Name:         "[Disabled] " + res.Data.Name,
		Description:  res.Data.Description,
		Version:      res.Data.Version,
		Resources:    []stremio.Resource{},
		Types:        []stremio.ContentType{},
		Catalogs:     []stremio.Catalog{},
		Background:   res.Data.Background,
		Logo:         res.Data.Logo,
		ContactEmail: res.Data.ContactEmail,
	}

	return manifest, nil
}
