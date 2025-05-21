package stremio_sidekick

import (
	"bytes"
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var ExtractRequestBaseURL = shared.ExtractRequestBaseURL

func SendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	stremio_shared.SendResponse(w, r, statusCode, data)
}

func SendHTML(w http.ResponseWriter, statusCode int, data bytes.Buffer) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	stremio_shared.SendHTML(w, statusCode, data)
}

func hasCatalogBoard(catalog *stremio.Catalog) bool {
	for _, extra := range catalog.Extra {
		if extra.IsRequired {
			return false
		}
	}
	if len(catalog.ExtraRequired) > 0 {
		return false
	}
	return true

}

func canToggleCatalogBoard(catalog *stremio.Catalog) bool {
	for _, extra := range catalog.Extra {
		if extra.Name == "genre" || extra.Name == "search" {
			return true
		}
	}
	for _, extra := range catalog.ExtraSupported {
		if extra == "genre" || extra == "search" {
			return true
		}
	}
	return false
}

func toggleCatalogBoard(catalog *stremio.Catalog, hidden bool) {
	if hasCatalogBoard(catalog) == !hidden {
		return
	}

	toggledField := ""
	for i := range catalog.Extra {
		extra := &catalog.Extra[i]
		if extra.Name == "genre" {
			extra.IsRequired = hidden
			toggledField = extra.Name
		}
	}
	if toggledField == "" {
		for i := range catalog.Extra {
			extra := &catalog.Extra[i]
			if extra.Name == "search" {
				extra.IsRequired = hidden
				toggledField = extra.Name
			}
		}
	}

	if toggledField == "" {
		return
	}

	if len(catalog.ExtraSupported) > 0 {
		if hidden {
			if slices.Contains(catalog.ExtraSupported, toggledField) && !slices.Contains(catalog.ExtraRequired, toggledField) {
				catalog.ExtraRequired = append(catalog.ExtraRequired, toggledField)
			}
		} else {
			if slices.Contains(catalog.ExtraRequired, toggledField) {
				catalog.ExtraRequired = slices.DeleteFunc(catalog.ExtraRequired, func(name string) bool {
					return name == toggledField
				})
			}
		}
	}
}
