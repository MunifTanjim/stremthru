package endpoint

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func handleExperimentZileanTorrents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	q := r.URL.Query()
	noApproxSize := q.Get("no_approx_size") != ""
	noMissingSize := q.Get("no_missing_size") != ""
	excludeSource := strings.Split(q.Get("exclude_source"), ",")

	items, err := ti.DumpTorrents(noApproxSize, noMissingSize, excludeSource)
	if err != nil {
		SendError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(items); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

type SabnzbdErrorResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

func handleSabnzbdAPI(w http.ResponseWriter, r *http.Request) {
	rCtx := server.GetReqCtx(r)
	rCtx.RedactURLQueryParams(r, "apikey")

	q := r.URL.Query()

	apikey := q.Get("apikey")
	if apikey == "" {
		shared.SendHTML(w, http.StatusForbidden, *bytes.NewBuffer([]byte("API Key Required")))
		return
	}

	user := config.SabnzbdAuth.GetUser(apikey)
	if user == "" {
		shared.SendHTML(w, http.StatusForbidden, *bytes.NewBuffer([]byte("API Key Incorrect")))
		return
	}

	mode := q.Get("mode")

	switch mode {
	case "addurl":
		handleSabnzbdAddUrl(w, r, user)
	default:
		shared.SendJSON(w, r, http.StatusOK, SabnzbdErrorResponse{
			Status: false,
			Error:  "not implemented",
		})
	}
}

type SabnzbdAddUrlResponse struct {
	Status bool     `json:"status"`
	NzoIds []string `json:"nzo_ids"`
}

func handleSabnzbdAddUrl(w http.ResponseWriter, r *http.Request, user string) {
	log := server.GetReqCtx(r).Log

	q := r.URL.Query()

	nzbURL := q.Get("name")
	if nzbURL == "" {
		shared.SendJSON(w, r, http.StatusBadRequest, SabnzbdErrorResponse{
			Status: false,
			Error:  "expects one parameter",
		})
		return
	}

	nzbName := q.Get("nzbname")

	category := q.Get("cat")
	if category == "*" {
		category = ""
	}

	priority := util.SafeParseInt(q.Get("priority"), 0)
	if priority == -100 {
		priority = 0
	}

	password := q.Get("password")

	id, err := nzb_info.QueueJob(user, nzbName, nzbURL, category, priority, password)
	if err != nil {
		log.Error("failed to insert sabnzbd nzb queue item", "error", err)
		shared.SendHTML(w, http.StatusInternalServerError, *bytes.NewBuffer([]byte("Internal Server Error")))
		return
	}

	shared.SendJSON(w, r, http.StatusOK, SabnzbdAddUrlResponse{
		Status: true,
		NzoIds: []string{"SABnzbd_nzo_" + id},
	})
}

func AddExperimentEndpoints(mux *http.ServeMux) {
	withAdminAuth := server.Middleware(server.AdminAuthed)

	mux.HandleFunc("/__experiment__/zilean/torrents", withAdminAuth(handleExperimentZileanTorrents))

	if config.Feature.HasVault() {
		mux.HandleFunc("/v0/__experiment__/sabnzbd/api", handleSabnzbdAPI)
	}
}
