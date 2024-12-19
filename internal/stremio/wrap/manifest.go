package stremio_wrap

import (
	"github.com/MunifTanjim/stremthru/stremio"
)

func getManifest(upstream *stremio.Manifest, ud *UserData) *stremio.Manifest {
	manifest := upstream
	manifest.ID = "st:wrap:" + manifest.ID
	manifest.Name = "StremThru(" + manifest.Name + ")"
	manifest.BehaviorHints = &stremio.BehaviorHints{
		Configurable:          true,
		ConfigurationRequired: !ud.HasRequiredValues(),
	}
	for i := range manifest.Resources {
		r := &manifest.Resources[i]
		if r.Name == stremio.ResourceNameCatalog && len(r.Types) == 0 {
			r.Types = manifest.Types
		}
	}
	return manifest
}
