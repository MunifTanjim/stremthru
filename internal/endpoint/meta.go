package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/meta"
)

func AddMetaEndpoints(mux *http.ServeMux) {
	meta.AddEndpoints(mux)
}
