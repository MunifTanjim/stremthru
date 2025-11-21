package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/job_log"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/worker"
)

type WorkerDetails struct {
	Id           string        `json:"id"`
	Title        string        `json:"title"`
	Interval     time.Duration `json:"interval"`
	HasFailedJob bool          `json:"has_failed_job"`
}

func handleGetWorkersDetails(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	data := make(map[string]*WorkerDetails, len(worker.WorkerDetailsById))

	for name, details := range worker.WorkerDetailsById {
		data[name] = &WorkerDetails{
			Id:       details.Id,
			Title:    details.Title,
			Interval: details.Interval,
		}
	}

	failedWorkerNames, err := job_log.GetWorkerNamesWithFailedJobs()
	if err != nil {
		SendError(w, r, err)
		return
	}

	for _, workerName := range failedWorkerNames {
		if workerResp, ok := data[workerName]; ok {
			workerResp.HasFailedJob = true
		}
	}

	SendData(w, r, 200, data)
}

func handleGetWorkerJobLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	if _, ok := worker.WorkerDetailsById[name]; !ok {
		ErrorBadRequest(r, "invalid worker id").Send(w, r)
		return
	}

	jobLogs, err := job_log.GetAllJobLogs[any](name)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, jobLogs)
}

func handlePurgeWorkerJobLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	if _, ok := worker.WorkerDetailsById[name]; !ok {
		ErrorBadRequest(r, "invalid worker id").Send(w, r)
		return
	}

	err := job_log.PurgeJobLogs(name)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 204, nil)
}

func handleWorkerJobLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetWorkerJobLogs(w, r)
	case http.MethodDelete:
		handlePurgeWorkerJobLogs(w, r)
	default:
		ErrorMethodNotAllowed(r).Send(w, r)
	}
}

func AddWorkerEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/workers/details", authed(handleGetWorkersDetails))
	router.HandleFunc("/workers/{id}/job-logs", authed(handleWorkerJobLogs))
}
