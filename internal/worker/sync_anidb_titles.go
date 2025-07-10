package worker

import (
	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncAniDBTitlesJobTracker *JobTracker[struct{}]

func isAnidbTitlesSyncedToday() bool {
	if syncAniDBTitlesJobTracker == nil {
		return false
	}
	job, err := syncAniDBTitlesJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && util.IsToday(job.CreatedAt) && job.Value.Status == "done"
}

func InitSyncAniDBTitlesWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		err := anidb.SyncTitleDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	if worker != nil {
		syncAniDBTitlesJobTracker = worker.jobTracker
	}

	return worker
}
