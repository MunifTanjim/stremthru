package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
)

var syncIMDBJobTracker *JobTracker[struct{}]

func getTodayDateOnly() string {
	return time.Now().Format(time.DateOnly)
}

func isIMDBSyncedToday() bool {
	if syncIMDBJobTracker == nil {
		return false
	}
	jobId := getTodayDateOnly()
	job, err := syncIMDBJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncIMDBWorker(conf *WorkerConfig) *Worker {
	syncIMDBJobTracker = conf.JobTracker

	conf.Executor = func(w *Worker) error {
		if err := imdb_title.SyncDataset(); err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	return worker
}
