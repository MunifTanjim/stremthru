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
	"github.com/MunifTanjim/stremthru/store"
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

	id, err := nzb_info.QueueJob(user, nzbName, nzbURL, category, priority, password, 0)
	if err != nil {
		log.Error("failed to insert sabnzbd nzb queue item", "error", err)
		shared.SendHTML(w, http.StatusInternalServerError, *bytes.NewBuffer([]byte("Internal Server Error")))
		return
	}

	shared.SendJSON(w, r, http.StatusOK, SabnzbdAddUrlResponse{
		Status: true,
		NzoIds: []string{nzoIDPrefix +id},
	})
}

func handleSabnzbdHistory(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	infos, err := nzb_info.GetAll()
	if err != nil {
		log.Error("failed to get nzb history", "error", err)
		shared.SendJSON(w, r, http.StatusInternalServerError, SabnzbdErrorResponse{
			Status: false,
			Error:  "failed to get history",
		})
		return
	}

	slots := make([]map[string]any, 0, len(infos))
	for _, info := range infos {
		var status string
		completed := int64(0)
		downloaded := int64(0)
		switch store.NewzStatus(info.Status) {
		case store.NewzStatusDownloaded, store.NewzStatusCached:
			status = "Completed"
			completed = info.UAt.Unix()
			downloaded = info.Size
		case store.NewzStatusFailed, store.NewzStatusInvalid:
			status = "Failed"
		default:
			continue
		}

		slots = append(slots, map[string]any{
			"completed":  completed,
			"downloaded": downloaded,
			"name":       info.Name,
			"nzo_id":     nzoIDPrefix +info.Hash,
			"status":     status,
		})
	}

	shared.SendJSON(w, r, http.StatusOK, map[string]any{
		"history": map[string]any{
			"slots":   slots,
			"version": version,
		},
	})
}

func handleSabnzbdQueue(w http.ResponseWriter, r *http.Request, user string) {
	log := server.GetReqCtx(r).Log

	jobs, err := nzb_info.GetAllJob()
	if err != nil {
		log.Error("failed to get nzb queue", "error", err)
		shared.SendJSON(w, r, http.StatusInternalServerError, SabnzbdErrorResponse{
			Status: false,
			Error:  "failed to get queue",
		})
		return
	}

	hashes := make([]string, len(jobs))
	for i, job := range jobs {
		hashes[i] = job.Key
	}
	infoByHash, err := nzb_info.GetByHashes(hashes)
	if err != nil {
		log.Warn("failed to get nzb info by hashes", "error", err)
		infoByHash = map[string]*nzb_info.NZBInfo{}
	}

	slots := make([]map[string]any, 0, len(jobs))
	queueStatus := "Idle"
	var totalMbLeft float64
	for i, job := range jobs {
		filename := job.Payload.Data.Name
		mbLeft := "0.00"
		percentage := "0"

		info := infoByHash[job.Key]

		status := ""
		switch job_queue.EntryStatus(job.Status) {
		case job_queue.EntryStatusQueued:
			status = "Queued"
		case job_queue.EntryStatusProcessing:
			status = "Downloading"
			queueStatus = "Downloading"
		case job_queue.EntryStatusFailed:
			status = "Queued"
		case job_queue.EntryStatusDead:
			status = "Failed"
		case job_queue.EntryStatusDone:
			if info == nil {
				status = "Failed"
			} else if info.Streamable {
				status = "Completed"
				percentage = "100"
			} else {
				status = "Failed"
			}
		}

		if info != nil {
			if info.Name != "" {
				filename = info.Name
			}
			if status == "Queued" || status == "Downloading" {
				mb := float64(info.Size) / 1024 / 1024
				totalMbLeft += mb
				mbLeft = fmt.Sprintf("%.2f", mb)
			}
		}

		slots = append(slots, map[string]any{
			"filename":   filename,
			"mbleft":     mbLeft,
			"nzo_id":     nzoIDPrefix +job.Key,
			"percentage": percentage,
			"status":     status,
			"timeleft":   nil,

			"cat":        job.Payload.Data.Category,
			"index":      i,
			"password":   job.Payload.Data.Password,
			"time_added": job.CreatedAt.Unix(),
		})
	}

	shared.SendJSON(w, r, http.StatusOK, map[string]any{
		"queue": map[string]any{
			"kbpersec": nil,
			"mbleft":   fmt.Sprintf("%.2f", totalMbLeft),
			"paused":   false,
			"slots":    slots,
			"status":   queueStatus,
			"timeleft": nil,
			"version":  version,
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
	case "history":
		handleSabnzbdHistory(w, r)
	case "get_cats":
		cats := make([]string, 0, len(categories))
		for _, c := range categories {
			cats = append(cats, c["name"].(string))
		}
		shared.SendJSON(w, r, http.StatusOK, map[string]any{
			"categories": cats,
		})
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
	if !config.Feature.HasNewz() || !config.Feature.HasVault() {
		return
	}

	mux.HandleFunc("/v0/sabnzbd/api", handleSabnzbdAPI)
}
