package worker

import (
	"github.com/MunifTanjim/stremthru/internal/anidb"
)

var syncAniDBTitlesJobTracker *JobTracker[struct{}]

func isAnidbTitlesSynced() bool {
	if syncAniDBTitlesJobTracker == nil {
		return false
	}
	jobId := getTodayDateOnly()
	job, err := syncAniDBTitlesJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncAniDBTitlesWorker(conf *WorkerConfig) *Worker {
	syncAniDBTitlesJobTracker = conf.JobTracker

	conf.Executor = func(w *Worker) error {
		err := anidb.SyncTitleDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	return worker
}
