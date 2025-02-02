package stremio_wrap

import (
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

func getManifestResourceTypes(manifest *stremio.Manifest, resource stremio.Resource) []stremio.ContentType {
	if len(resource.Types) == 0 {
		return manifest.Types
	}
	return resource.Types
}

func getManifestResourceIdPrefixes(manifest *stremio.Manifest, resource stremio.Resource) []string {
	if len(resource.IDPrefixes) == 0 {
		return manifest.IDPrefixes
	}
	return resource.IDPrefixes
}

func getManifest(upstreamManifests []stremio.Manifest, ud *UserData) *stremio.Manifest {
	manifest := stremio.Manifest{
		Version:   config.Version,
		Resources: []stremio.Resource{},
		Types:     []stremio.ContentType{},
		Catalogs:  []stremio.Catalog{},
	}

	id := ""
	name := ""
	description := ""
	for i := range upstreamManifests {
		if i > 0 {
			id += "::"
			description += "\n\n"
		}
		m := upstreamManifests[i]
		id += m.ID
		name += "(" + m.Name + ")"

		description += "[" + m.Name + " v" + m.Version + "]" + "(" + ud.Upstreams[i].URL + ")"
	}

	manifest.ID = "st:wrap::" + id

	storeHint := ""
	if ud.StoreName == "" {
		storeHint = "(ST)"
	} else {
		storeHint = "(" + strings.ToUpper(string(store.StoreName(ud.StoreName).Code())) + ")"
	}

	manifest.Name = "StremThru" + storeHint + name
	manifest.Description = description
	manifest.BehaviorHints = &stremio.BehaviorHints{
		Configurable:          true,
		ConfigurationRequired: !ud.HasRequiredValues(),
	}

	resourceByName := map[stremio.ResourceName]stremio.Resource{}
	typesMap := map[stremio.ResourceName]map[stremio.ContentType]bool{}
	idPrefixesMap := map[stremio.ResourceName]map[string]bool{}

	for mIdx := range upstreamManifests {
		m := upstreamManifests[mIdx]
		for _, r := range m.Resources {
			if _, found := resourceByName[r.Name]; !found {
				resourceByName[r.Name] = stremio.Resource{Name: r.Name}
				typesMap[r.Name] = map[stremio.ContentType]bool{}
				idPrefixesMap[r.Name] = map[string]bool{}
			}

			tMap := typesMap[r.Name]
			for _, t := range getManifestResourceTypes(&m, r) {
				tMap[t] = true
			}

			switch r.Name {
			case stremio.ResourceNameAddonCatalog:
				for i := range m.AddonCatalogs {
					ac := m.AddonCatalogs[i]
					ac.Id = strconv.Itoa(mIdx) + "::" + ac.Id
					manifest.AddonCatalogs = append(manifest.AddonCatalogs, ac)
				}
			case stremio.ResourceNameCatalog:
				for i := range m.Catalogs {
					c := m.Catalogs[i]
					c.Id = strconv.Itoa(mIdx) + "::" + c.Id
					manifest.Catalogs = append(manifest.Catalogs, c)
				}
			default:
				idpMap := idPrefixesMap[r.Name]
				for _, p := range getManifestResourceIdPrefixes(&m, r) {
					idpMap[p] = true
				}
			}
		}
	}

	for rName := range resourceByName {
		r := resourceByName[rName]

		if rName != stremio.ResourceNameAddonCatalog {
			types := []stremio.ContentType{}
			for cType := range typesMap[rName] {
				types = append(types, cType)
			}
			r.Types = types

			if rName != stremio.ResourceNameCatalog {
				idPrefixes := []string{}
				for idPrefix := range idPrefixesMap[rName] {
					idPrefixes = append(idPrefixes, idPrefix)
				}
				r.IDPrefixes = idPrefixes
			}
		}

		manifest.Resources = append(manifest.Resources, r)
	}

	return &manifest
}
