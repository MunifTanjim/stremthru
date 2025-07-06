package worker

import (
	"github.com/MunifTanjim/stremthru/internal/manami"
)

var syncManamiAnimeDatabaseJobTracker *JobTracker[struct{}]

func isManamiAnimeDatabaseSynced() bool {
	if syncManamiAnimeDatabaseJobTracker == nil {
		return false
	}
	jobId := getTodayDateOnly()
	job, err := syncManamiAnimeDatabaseJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncManamiAnimeDatabaseWorker(conf *WorkerConfig) *Worker {
	syncManamiAnimeDatabaseJobTracker = conf.JobTracker

	conf.Executor = func(w *Worker) error {
		err := manami.SyncDataset()
		if err != nil {
			return err
		}
		return nil

	}

	worker := NewWorker(conf)

	return worker
}
