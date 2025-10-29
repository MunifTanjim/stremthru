package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncIMDBJobTracker *JobTracker[struct{}]

func isIMDBSyncedInLast24Hours() bool {
	if syncIMDBJobTracker == nil {
		return false
	}
	job, err := syncIMDBJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done" && !util.HasDurationPassedSince(job.CreatedAt, 24*time.Hour)
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
