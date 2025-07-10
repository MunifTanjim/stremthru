package worker

import (
	"github.com/MunifTanjim/stremthru/internal/animeapi"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncAnimeAPIJobTracker *JobTracker[struct{}]

func isAnimeAPISyncedToday() bool {
	if syncAnimeAPIJobTracker == nil {
		return false
	}
	job, err := syncAnimeAPIJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && util.IsToday(job.CreatedAt) && job.Value.Status == "done"
}

func InitSyncAnimeAPIWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		err := animeapi.SyncDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	if worker != nil {
		syncAnimeAPIJobTracker = worker.jobTracker
	}

	return worker
}
