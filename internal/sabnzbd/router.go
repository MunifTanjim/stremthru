package sabnzbd

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
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

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func mapPriority(priority int) string {
	switch {
	case priority < 0:
		return "Low"
	case priority > 0:
		return "High"
	default:
		return "Normal"
	}
}

func handleSabnzbdQueue(w http.ResponseWriter, r *http.Request, user string) {
	log := server.GetReqCtx(r).Log

	jobs, err := nzb_info.GetActiveJobs(3)
	if err != nil {
		log.Error("failed to get nzb queue", "error", err)
		shared.SendJSON(w, r, http.StatusInternalServerError, SabnzbdErrorResponse{
			Status: false,
			Error:  "failed to get queue",
		})
		return
	}

	slots := []map[string]any{}
	for i, job := range jobs {
		status := "Queued"
		if job.Status == string(job_queue.EntryStatusProcessing) {
			status = "Downloading"
		}

		filename := job.Payload.Data.Name
		var mb string
		var size string

		info, err := nzb_info.GetByHash(job.Key)
		if err != nil {
			log.Warn("failed to get nzb info", "hash", job.Key, "error", err)
		}
		if info != nil {
			if info.Name != "" {
				filename = info.Name
			}
			mb = fmt.Sprintf("%.2f", float64(info.Size)/1024/1024)
			size = formatBytes(info.Size)
		}

		slots = append(slots, map[string]any{
			"status":        status,
			"index":         i,
			"password":      job.Payload.Data.Password,
			"avg_age":       "",
			"time_added":    job.CreatedAt.Unix(),
			"script":        "None",
			"direct_unpack": nil,
			"mb":            mb,
			"mbleft":        mb,
			"mbmissing":     "0.0",
			"size":          size,
			"sizeleft":      size,
			"labels":        []any{},
			"priority":      mapPriority(job.Payload.Data.Priority),
			"cat":           job.Payload.Data.Category,
			"nzo_id":        "SABnzbd_nzo_" + job.Key,
			"unpackopts":    "3",
			"filename":      filename,
			"timeleft":      "0:00:00",
			"percentage":    "0",
		})
	}

	shared.SendJSON(w, r, http.StatusOK, map[string]any{
		"queue": map[string]any{
			"paused":          false,
			"slots":           slots,
			"version":         version,
			"paused_all":      false,
			"have_quota":      false,
			"noofslots_total": len(slots),
			"noofslots":       len(slots),
		},
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
	case "get_config":
		host := r.URL.Hostname()
		if host == "" {
			host = config.BaseURL.Hostname()
		}
		port := r.URL.Port()
		if port == "" {
			port = config.BaseURL.Port()
		}
		shared.SendJSON(w, r, http.StatusOK, map[string]any{
			"config": map[string]any{
				"misc": map[string]any{
					"host":     host,
					"port":     port,
					"username": "",
					"password": "",
					"api_key":  apikey,
					"nzb_key":  apikey,
					"url_base": strings.TrimSuffix(strings.Trim(r.URL.Path, "/"), "/api"),
				},
				"logging": map[string]any{
					"log_level":    1,
					"max_log_size": 5242880,
					"log_backups":  5,
				},
				"categories": categories,
				"servers":    servers,
			},
		})
	case "fullstatus", "status":
		shared.SendJSON(w, r, http.StatusOK, map[string]any{
			"status": map[string]any{
				"url_base": strings.TrimSuffix(strings.Trim(r.URL.Path, "/"), "/api"),
				"apikey":   apikey,
				"version":  version,
				"servers":  servers,
			},
		})
	case "queue":
		handleSabnzbdQueue(w, r, user)
	case "version":
		shared.SendJSON(w, r, http.StatusOK, map[string]string{
			"version": version,
		})
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
