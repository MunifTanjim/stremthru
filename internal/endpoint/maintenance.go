package endpoint

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/server"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

//go:embed maintenance.html
var maintenanceTemplateBlob string

type maintenanceTemplateData struct {
	Description string
	EndsAtUnix  int64
}

var executeMaintenanceTemplate = func() func(data *maintenanceTemplateData) (bytes.Buffer, error) {
	tmpl := template.Must(template.New("maintenance.html").Parse(maintenanceTemplateBlob))
	return func(data *maintenanceTemplateData) (bytes.Buffer, error) {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		return buf, err
	}
}()

func handleMaintenance(w http.ResponseWriter, r *http.Request) {
	td := &maintenanceTemplateData{}

	endsAt := server.GetMaintenanceEndTime()
	if endsAt.After(time.Now().Truncate(time.Second)) {
		td.Description = stremio_shared.GetMaintenanceDescription()
		td.EndsAtUnix = endsAt.Unix()
	}

	buf, err := executeMaintenanceTemplate(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, buf)
}

func AddMaintenanceEndpoint(mux *http.ServeMux) {
	mux.HandleFunc("GET /maintenance", handleMaintenance)
}
