package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncIMDBJobTracker *JobTracker[struct{}]

func getTodayDateOnly() string {
	return time.Now().Format(time.DateOnly)
}

func isIMDBSyncedToday() bool {
	if syncIMDBJobTracker == nil {
		return false
	}
	job, err := syncIMDBJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && util.IsToday(job.CreatedAt) && job.Value.Status == "done"
}

func InitSyncIMDBWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		if err := imdb_title.SyncDataset(); err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	if worker != nil {
		syncIMDBJobTracker = worker.jobTracker
	}

	return worker
}
