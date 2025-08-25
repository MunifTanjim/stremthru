package endpoint

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/meta"
	meta_list "github.com/MunifTanjim/stremthru/internal/meta/list"
)

func AddMetaEndpoints(mux *http.ServeMux) {
	meta.AddEndpoints(mux)
	meta_list.AddEndpoints(mux)
}
