package stremio_wrap

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

const catalog_id_calendar_videos = "calendar-videos"

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

func GetManifest(r *http.Request, upstreamManifests []stremio.Manifest, ud *UserData) *stremio.Manifest {
	manifest := &stremio.Manifest{
		Version:   config.Version,
		Logo:      "https://emojiapi.dev/api/v1/sparkles/256.png",
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
			description += "  \n\n"
		}
		m := upstreamManifests[i]
		id += m.ID
		name += "(" + m.Name + ")"

		hostname := ""
		if upUrl, err := url.Parse(ud.Upstreams[i].URL); err == nil {
			hostname = upUrl.Host
		} else {
			hostname, _, _ = strings.Cut(strings.TrimPrefix(strings.TrimPrefix(ud.Upstreams[i].URL, "http://"), "https://"), "/")
		}
		description += "[" + m.Name + " v" + m.Version + "]" + "(" + hostname + ")"
	}

	manifest.ID = "st:wrap::" + id

	storeHint := ""
	for i := range ud.Stores {
		code := string(ud.Stores[i].Code)
		if code == "" {
			code = "st"
		}
		if i > 0 {
			storeHint += "|"
		}
		storeHint += code
	}
	if storeHint != "" {
		storeHint = "(" + strings.ToUpper(storeHint) + ")"
	}

	manifest.Name = "StremThru" + storeHint + name
	manifest.Description = description
	manifest.BehaviorHints = &stremio.BehaviorHints{
		Configurable:          true,
		ConfigurationRequired: !ud.HasRequiredValues(),
	}

	if len(upstreamManifests) == 0 {
		manifest.ID = shared.GetReversedHostname(r) + ".wrap"
		manifest.Name = "StremThru Wrap"
		manifest.Description = "Stremio Addon to Wrap other Addons with StremThru"

		stremio_shared.ClaimAddonOnStremioAddonsDotNet(manifest, "eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4Q0JDLUhTMjU2In0..TZNrt5BpBCR1Qt-EpRoLBg.5OIDlcyBZDC6KG2gqXLi-A9beJhl2mvKZoiMQ8MlBNqzowgd-jMtu5WLRBUqFfuHemkItmqWNnTV17JktfJNFFl1ookUSHM8LUoezTOCBBTCOAWVJpbCy8dKs_lo5Bpa.j9_jgJc8T32v74o4zVMu8g")
	}

	if len(upstreamManifests) == 1 {
		manifest.Background = upstreamManifests[0].Background
		manifest.ContactEmail = upstreamManifests[0].ContactEmail
		manifest.Description = upstreamManifests[0].Description
		manifest.Logo = upstreamManifests[0].Logo
		manifest.Version = upstreamManifests[0].Version

		if len(upstreamManifests[0].AddonCatalogs) > 0 {
			manifest.AddonCatalogs = upstreamManifests[0].AddonCatalogs
		}
		if len(upstreamManifests[0].Catalogs) > 0 {
			manifest.Catalogs = upstreamManifests[0].Catalogs
		}
		if len(upstreamManifests[0].Types) > 0 {
			manifest.Types = upstreamManifests[0].Types
		}
		if len(upstreamManifests[0].IDPrefixes) > 0 {
			manifest.IDPrefixes = upstreamManifests[0].IDPrefixes
		}
		manifest.Resources = upstreamManifests[0].Resources

		if upstreamManifests[0].BehaviorHints != nil && upstreamManifests[0].BehaviorHints.NewEpisodeNotifications {
			manifest.BehaviorHints.NewEpisodeNotifications = true
		}

		return manifest
	}

	resourceByName := map[stremio.ResourceName]stremio.Resource{}
	typesMap := map[stremio.ResourceName]map[stremio.ContentType]bool{}
	idPrefixesMap := map[stremio.ResourceName]map[string]bool{}

	var calendarVideosCatalog *stremio.Catalog

	for mIdx := range upstreamManifests {
		m := upstreamManifests[mIdx]
		for _, r := range m.Resources {
			if IsPublicInstance {
				if r.Name == stremio.ResourceNameMeta || r.Name == stremio.ResourceNameSubtitles {
					continue
				}
			}

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
					switch c := m.Catalogs[i]; c.Id {
					case catalog_id_calendar_videos:
						if calendarVideosCatalog == nil {
							calendarVideosCatalog = &c
							manifest.Catalogs = append(manifest.Catalogs, c)
						}
					default:
						c.Id = strconv.Itoa(mIdx) + "::" + c.Id
						manifest.Catalogs = append(manifest.Catalogs, c)
					}
				}
			default:
				idpMap := idPrefixesMap[r.Name]
				for _, p := range getManifestResourceIdPrefixes(&m, r) {
					idpMap[p] = true
				}
			}
		}

		if m.BehaviorHints != nil && m.BehaviorHints.NewEpisodeNotifications {
			manifest.BehaviorHints.NewEpisodeNotifications = true
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

	return manifest
}
