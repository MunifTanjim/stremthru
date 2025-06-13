package worker

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/animelists"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

func InitSyncAniDBTVDBEpisodeMapWorker(conf *WorkerConfig) *Worker {
	if !config.Feature.IsEnabled("anime") {
		return nil
	}

	log := logger.Scoped("worker/sync_anidb_tvdb_episode_map")

	jobTracker := NewJobTracker("sync-anidb-tvdb-episode-map", func(id string, job *Job[struct{}]) bool {
		date, err := time.Parse(time.DateOnly, id)
		if err != nil {
			return true
		}
		return date.Before(time.Now().Add(-7 * 24 * time.Hour))
	})

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	jobId := ""
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(1 * 24 * time.Hour),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else if err == nil {
					jobId = ""
				}
				worker.onEnd()
			}()

			for {
				wait, reason := worker.shouldWait()
				if !wait {
					break
				}
				log.Info("waiting, " + reason)
				time.Sleep(5 * time.Minute)
			}
			worker.onStart()

			if jobId != "" {
				return nil
			}

			jobId = getTodayDateOnly()

			job, err := jobTracker.Get(jobId)
			if err != nil {
				return err
			}

			if job != nil && (job.Status == "done" || job.Status == "started") {
				log.Info("already done or started", "jobId", jobId, "status", job.Status)
				return nil
			}

			err = jobTracker.Set(jobId, "started", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "started")
				return err
			}

			err = animelists.SyncDataset()
			if err != nil {
				return err
			}

			err = jobTracker.Set(jobId, "done", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "done")
				return err
			}

			log.Info("done", "date", jobId)

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			if terr := jobTracker.Set(jobId, "failed", err.Error(), nil); terr != nil {
				log.Error("failed to set job status", "error", terr, "jobId", jobId, "status", "failed")
			}

			jobId = ""
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	if task, err := worker.scheduler.Lookup(id); err == nil && task != nil {
		t := task.Clone()
		t.Interval = 30 * time.Second
		t.RunOnce = true
		worker.scheduler.Add(t)
	}

	return worker
}
