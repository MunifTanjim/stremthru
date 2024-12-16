package stremio_store

import "github.com/MunifTanjim/stremthru/stremio"

const CATALOG_ID = "st:store"
const ID_PREFIX = CATALOG_ID + ":"
const STORE_ACTION_ID = ID_PREFIX + "action"
const STORE_ACTION_ID_PREFIX = STORE_ACTION_ID + ":"

const ContentTypeOther = "other"

const (
	CatalogTypeVideo  = "video"
	CatalogTypeAction = "action"
)

func getManifest(ud *UserData) *stremio.Manifest {
	manifest := &stremio.Manifest{
		ID:          "dev.muniftanjim.stremthru.store",
		Name:        "Store",
		Description: "StremThru Store Catalog",
		Version:     "0.0.0", // x-release-please-version
		Resources: []stremio.Resource{
			stremio.Resource{
				Name:       stremio.ResourceNameMeta,
				Types:      []stremio.ContentType{ContentTypeOther},
				IDPrefixes: []string{ID_PREFIX},
			}, stremio.Resource{
				Name:       stremio.ResourceNameStream,
				Types:      []stremio.ContentType{ContentTypeOther},
				IDPrefixes: []string{ID_PREFIX},
			},
		},
		Catalogs: []stremio.Catalog{
			stremio.Catalog{
				Id:   CATALOG_ID,
				Name: "Store",
				Type: ContentTypeOther,
				Extra: []stremio.CatalogExtra{
					stremio.CatalogExtra{
						Name: "search",
					},
					stremio.CatalogExtra{
						Name: "skip",
					},
					stremio.CatalogExtra{
						Name:    "type",
						Options: []string{CatalogTypeVideo, CatalogTypeAction},
					},
				},
			},
		},
		ContactEmail: "github@muniftanjim.dev",
		BehaviorHints: &stremio.BehaviorHints{
			Configurable:          true,
			ConfigurationRequired: !ud.HasRequiredValues(),
		},
	}

	return manifest
}
