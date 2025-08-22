package stremio_meta

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var ExtractRequestBaseURL = shared.ExtractRequestBaseURL

var SendResponse = stremio_shared.SendResponse
var SendHTML = stremio_shared.SendHTML
var GetPathValue = stremio_shared.GetPathValue

func getContentType(r *http.Request) (string, error) {
	contentType := r.PathValue("contentType")
	return contentType, nil
}

func getId(r *http.Request) string {
	return GetPathValue(r, "id")
}
