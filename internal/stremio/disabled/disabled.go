package stremio_disabled

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/shared"
)

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	manifest, err := GetDisabledManifest(r.PathValue("manifestUrl"))
	if err != nil {
		shared.SendError(w, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(manifest); err != nil {
		log.Printf("failed to encode json %v\n", err)
	}
}

func AddStremioDisabledEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/stremio/disabled/{manifestUrl}/manifest.json", handleManifest)
}
