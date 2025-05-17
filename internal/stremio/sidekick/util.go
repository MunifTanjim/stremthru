package stremio_sidekick

import (
	"bytes"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
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
