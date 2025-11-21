package dash_api

import (
	"errors"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/job_log"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
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

type WorkerTemporaryFile struct {
	Path       string `json:"path"`
	Size       string `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

func handleGetWorkerTemporaryFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")

	files := []WorkerTemporaryFile{}
	switch name {
	case "sync-imdb":
		dirPath := imdb_title.GetDatasetTemporaryDir()
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err != nil {
					return err
				}
				files = append(files, WorkerTemporaryFile{
					Path:       strings.Replace(path, config.DataDir, "DATA_DIR", 1),
					Size:       util.ToSize(info.Size()),
					ModifiedAt: info.ModTime().Format(time.RFC3339),
				})
			}
			return nil
		})
		if err != nil {
			var perr *fs.PathError
			if !errors.As(err, &perr) || !strings.Contains(perr.Err.Error(), "no such file or directory") {
				SendError(w, r, err)
				return
			}
		}
		SendData(w, r, 200, files)
	default:
		if _, ok := worker.WorkerDetailsById[name]; ok {
			ErrorBadRequest(r, "worker does not support temporary files").Send(w, r)
		} else {
			ErrorBadRequest(r, "invalid worker id").Send(w, r)
		}
	}
}

func handlePurgeWorkerTemporaryFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	if _, ok := worker.WorkerDetailsById[name]; !ok {
		ErrorBadRequest(r, "invalid worker id").Send(w, r)
		return
	}

	switch name {
	case "sync-imdb":
		err := imdb_title.PurgeDatasetTemporaryFiles()
		if err != nil {
			if errors.Is(err, imdb_title.ErrDatasetSyncInProgress) {
				ErrorLocked(r, err.Error()).WithCause(err).Send(w, r)
			} else {
				SendError(w, r, err)
			}
			return
		}
		SendData(w, r, 204, nil)
	default:
		ErrorBadRequest(r, "worker does not support temporary file purge").Send(w, r)
		return
	}

}

func handleWorkerTemporaryFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetWorkerTemporaryFiles(w, r)
	case http.MethodDelete:
		handlePurgeWorkerTemporaryFiles(w, r)
	default:
		ErrorMethodNotAllowed(r).Send(w, r)
	}
}

func AddWorkerEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/workers/details", authed(handleGetWorkersDetails))
	router.HandleFunc("/workers/{id}/job-logs", authed(handleWorkerJobLogs))
	router.HandleFunc("/workers/{id}/temporary-files", authed(handleWorkerTemporaryFiles))
}
