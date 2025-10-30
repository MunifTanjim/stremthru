package dash_api

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/job_log"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/worker"
)

func HandleGetWorkersDetails(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	SendData(w, r, 200, worker.WorkerDetailsById)
}

func HandleGetWorkerJobLogs(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

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
