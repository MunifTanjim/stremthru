package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	meta_id_map "github.com/MunifTanjim/stremthru/internal/meta/id_map"
	meta_letterboxd "github.com/MunifTanjim/stremthru/internal/meta/letterboxd"
)

func AddMetaEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasMeta() {
		return
	}

	meta_id_map.AddEndpoints(mux)
	meta_letterboxd.AddEndpoints(mux)
}
