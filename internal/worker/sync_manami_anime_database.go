package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/manami"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncManamiAnimeDatabaseJobTracker *JobTracker[struct{}]

func isManamiAnimeDatabaseSyncedThisWeek() bool {
	if syncManamiAnimeDatabaseJobTracker == nil {
		return false
	}
	job, err := syncManamiAnimeDatabaseJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && !util.HasDurationPassedSince(job.CreatedAt, 7*24*time.Hour) && job.Value.Status == "done"
}

func InitSyncManamiAnimeDatabaseWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		err := manami.SyncDataset()
		if err != nil {
			return err
		}
		return nil

	}

	worker := NewWorker(conf)

	if worker != nil {
		syncManamiAnimeDatabaseJobTracker = worker.jobTracker
	}

	return worker
}
