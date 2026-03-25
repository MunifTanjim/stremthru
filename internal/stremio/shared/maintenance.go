package stremio_shared

import (
	"math/rand"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var maintenanceDescriptions = []string{
	"Even servers need a coffee break. Back before you finish saying 'buffering'!",
	"Server is hitting the gym. Lifting heavy loads right now, check back soon.",
	"Shhh... the server is taking a power nap. It'll wake up refreshed shortly.",
	"Our hamsters are changing the wheel. Normal service resumes shortly.",
	"The server went to get milk. It promises it'll come back this time.",
	"Have you tried waiting and trying again? Because that's genuinely the fix here.",
	"Server is contemplating the meaning of uptime. Deep thoughts take time.",
	"Currently teaching the server new tricks. Old servers, new tricks — you know how it is.",
	"The server is on a lunch break. No, it didn't bring enough for everyone.",
	"Capacity reached. The server is doing its best, and its best needs a minute.",
}

func GetMaintenanceDescription() string {
	return maintenanceDescriptions[rand.Intn(len(maintenanceDescriptions))]
}

func HandleMaintenance(w http.ResponseWriter, r *http.Request) bool {
	if !server.IsMaintenanceActive() {
		return false
	}

	path := r.URL.Path

	var data any
	switch {
	case strings.Contains(path, "/catalog/"):
		data = stremio.CatalogHandlerResponse{Metas: []stremio.MetaPreview{{
			Id:          "st:maintenance",
			Type:        "other",
			Name:        "StremThru 🛠️ Maintenance Ongoing",
			Description: GetMaintenanceDescription(),
		}}}
	case strings.Contains(path, "/stream/"):
		data = stremio.StreamHandlerResponse{Streams: []stremio.Stream{{
			Name:        "StremThru 🛠️ Maintenance Ongoing",
			Description: GetMaintenanceDescription(),
			ExternalURL: shared.ExtractRequestBaseURL(r).JoinPath("/maintenance").String(),
		}}}
	default:
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	SendResponse(w, r, http.StatusOK, data)
	return true
}
