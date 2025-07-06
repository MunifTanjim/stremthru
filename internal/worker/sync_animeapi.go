package worker

import (
	"github.com/MunifTanjim/stremthru/internal/animeapi"
)

var syncAnimeAPIJobTracker *JobTracker[struct{}]

func isAnimeAPISynced() bool {
	if syncAnimeAPIJobTracker == nil {
		return false
	}
	jobId := getTodayDateOnly()
	job, err := syncAnimeAPIJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncAnimeAPIWorker(conf *WorkerConfig) *Worker {
	syncAnimeAPIJobTracker = conf.JobTracker

	conf.Executor = func(w *Worker) error {
		err := animeapi.SyncDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	return worker
}
