package sabnzbd

import (
	"bytes"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type SabnzbdErrorResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
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

func handleSabnzbdAPI(w http.ResponseWriter, r *http.Request) {
	rCtx := server.GetReqCtx(r)
	rCtx.RedactURLQueryParams(r, "apikey")

	q := r.URL.Query()

	apikey := q.Get("apikey")
	if apikey == "" {
		shared.SendHTML(w, http.StatusForbidden, *bytes.NewBuffer([]byte("API Key Required")))
		return
	}

	user := config.Auth.GetSABnzbdUser(apikey)
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

func AddEndpoints(mux *http.ServeMux) {
	if config.Feature.HasVault() {
		mux.HandleFunc("/v0/sabnzbd/api", handleSabnzbdAPI)
	}
}
