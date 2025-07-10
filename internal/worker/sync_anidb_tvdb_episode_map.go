package worker

import (
	"github.com/MunifTanjim/stremthru/internal/animelists"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var syncAniDBTVDBEpisodeMapJobTracker *JobTracker[struct{}]

func isAniDBTVDBEpisodeMapSyncedToday() bool {
	if syncAniDBTVDBEpisodeMapJobTracker == nil {
		return false
	}
	job, err := syncAniDBTVDBEpisodeMapJobTracker.GetLast()
	if err != nil {
		return false
	}
	return job != nil && util.IsToday(job.CreatedAt) && job.Value.Status == "done"
}

func InitSyncAniDBTVDBEpisodeMapWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		err := animelists.SyncDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	if worker != nil {
		syncAniDBTVDBEpisodeMapJobTracker = worker.jobTracker
	}

	return worker
}
