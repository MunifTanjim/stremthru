package stremio_shared

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/shared"
)

func SendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		LogError(r, "failed to encode json", err)
	}
}

var SendHTML = shared.SendHTML

func GetPathValue(r *http.Request, name string) string {
	if value := r.PathValue(name + "Json"); value != "" {
		return strings.TrimSuffix(value, ".json")
	}
	return r.PathValue(name)
}
