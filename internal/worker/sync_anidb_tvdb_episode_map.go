package worker

import (
	"github.com/MunifTanjim/stremthru/internal/animelists"
)

var syncAniDBTVDBEpisodeMapJobTracker *JobTracker[struct{}]

func isAniDBTVDBEpisodeMapSynced() bool {
	if syncAniDBTVDBEpisodeMapJobTracker == nil {
		return false
	}
	jobId := getTodayDateOnly()
	job, err := syncAniDBTVDBEpisodeMapJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncAniDBTVDBEpisodeMapWorker(conf *WorkerConfig) *Worker {
	syncAniDBTVDBEpisodeMapJobTracker = conf.JobTracker

	conf.Executor = func(w *Worker) error {
		err := animelists.SyncDataset()
		if err != nil {
			return err
		}
		return nil
	}

	worker := NewWorker(conf)

	return worker
}
