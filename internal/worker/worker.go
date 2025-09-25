package worker

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/kv"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/madflojo/tasks"
)

var mutex sync.Mutex
var running_worker struct {
	sync_anidb_titles           bool
	sync_bitmagnet              bool
	sync_dmm_hashlist           bool
	sync_imdb                   bool
	map_imdb_torrent            bool
	sync_animeapi               bool
	sync_anidb_tvdb_episode_map bool
	sync_manami_anime_database  bool
}

type Worker struct {
	scheduler  *tasks.Scheduler
	shouldWait func() (bool, string)
	onStart    func()
	onEnd      func()
	Log        *slog.Logger
	jobTracker *JobTracker[struct{}]
}

type WorkerConfig struct {
	Disabled          bool
	Executor          func(w *Worker) error
	Interval          time.Duration
	HeartbeatInterval time.Duration
	Log               *slog.Logger
	Name              string
	OnEnd             func()
	OnStart           func()
	RunAtStartupAfter time.Duration
	RunExclusive      bool
	ShouldWait        func() (bool, string)
}

func NewWorker(conf *WorkerConfig) *Worker {
	if conf.Name == "" {
		panic("worker name cannot be empty")
	}

	if conf.Disabled {
		return nil
	}

	if conf.Log == nil {
		conf.Log = logger.Scoped("worker/" + conf.Name)
	}

	if conf.HeartbeatInterval == 0 {
		conf.HeartbeatInterval = 5 * time.Second
	}
	heartbeatIntervalTolerance := min(conf.HeartbeatInterval, 10*time.Second)

	log := conf.Log

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
		Log:        log,
	}

	jobTrackerExpiresIn := max(3*24*time.Hour, 10*conf.Interval)
	jobTracker := NewJobTracker[struct{}](conf.Name, jobTrackerExpiresIn)
	worker.jobTracker = jobTracker

	jobId := ""
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          conf.Interval,
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			isAlreadyRunning := jobId != ""
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else if err == nil && !isAlreadyRunning {
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
				time.Sleep(1 * time.Minute)
			}
			worker.onStart()

			if isAlreadyRunning {
				return nil
			}

			lock := db.NewAdvisoryLock("worker", conf.Name)
			if lock == nil {
				log.Error("failed to create advisory lock", "name", conf.Name)
				return nil
			}

			if !lock.TryAcquire() {
				log.Debug("skipping, another instance is running", "name", lock.GetName())
				return nil
			}
			defer lock.Release()

			var tjob *kv.ParsedKV[Job[struct{}]]
			if conf.RunExclusive {
				tjob, err = jobTracker.GetLast()
				if err != nil {
					return err
				}
				if tjob != nil {
					status := tjob.Value.Status
					switch status {
					case "started":
						if !util.HasDurationPassedSince(tjob.UpdatedAt, conf.HeartbeatInterval+heartbeatIntervalTolerance) {
							if util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
								log.Warn("skipping, last job is still running, for too long", "jobId", tjob.Key, "status", status)
							} else {
								log.Info("skipping, last job is still running", "jobId", tjob.Key, "status", status)
							}
							return nil
						}

						log.Info("last job heartbeat timed out, restarting", "jobId", tjob.Key, "status", status)
						if err := jobTracker.Set(tjob.Key, "failed", "heartbeat timed out", nil); err != nil {
							log.Error("failed to set last job status", "error", err, "jobId", tjob.Key, "status", "failed")
						}
					case "done":
						if !util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
							log.Info("already done", "jobId", tjob.Key, "status", status)
							return nil
						}
					case "failed":
						log.Warn("last job failed", "jobId", tjob.Key, "status", status, "error", tjob.Value.Err)
					}
				}
			}

			jobId = time.Now().Format(time.DateTime)

			err = jobTracker.Set(jobId, "started", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "started")
				return err
			}

			if !lock.Release() {
				log.Error("failed to release advisory lock", "name", lock.GetName())
				return nil
			}

			heartbeat := time.NewTicker(conf.HeartbeatInterval)
			heartbeat_done := make(chan struct{})
			defer close(heartbeat_done)
			go func() {
				for {
					select {
					case <-heartbeat.C:
						if jobId == "" {
							return
						}
						if err := jobTracker.Set(jobId, "started", "", nil); err != nil {
							log.Error("failed to set job status heartbeat", "error", err, "jobId", jobId)
						}
					case <-heartbeat_done:
						heartbeat.Stop()
						return
					}
				}
			}()

			if err = conf.Executor(worker); err != nil {
				return err
			}

			err = jobTracker.Set(jobId, "done", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "done")
				return err
			}

			log.Info("done", "jobId", jobId)

			return err
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					log.Error("Worker Err Panic", "error", perr, "stack", stack)
				}
				jobId = ""
			}()

			if terr := jobTracker.Set(jobId, "failed", err.Error(), nil); terr != nil {
				log.Error("failed to set job status", "error", terr, "jobId", jobId, "status", "failed")
			}
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	if conf.RunAtStartupAfter != 0 {
		if task, err := worker.scheduler.Lookup(id); err == nil && task != nil {
			t := task.Clone()
			t.Interval = conf.RunAtStartupAfter
			t.RunOnce = true
			worker.scheduler.Add(t)
		}
	}

	return worker
}

func InitWorkers() func() {
	workers := []*Worker{}

	if worker := InitParseTorrentWorker(&WorkerConfig{
		Name:         "parse-torrent",
		Interval:     5 * time.Minute,
		RunExclusive: true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_bitmagnet {
				return true, "sync_bitmagnet is running"
			}
			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}
			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}
			if running_worker.map_imdb_torrent {
				return true, "map_imdb_torrent is running"
			}
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitPushTorrentsWorker(&WorkerConfig{
		Disabled: TorrentPusherQueue.disabled,
		Name:     "push-torrent",
		Interval: 10 * time.Minute,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitCrawlStoreWorker(&WorkerConfig{
		Name:     "crawl-store",
		Interval: 30 * time.Minute,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()
			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}
			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}
			if running_worker.map_imdb_torrent {
				return true, "map_imdb_torrent is running"
			}
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncIMDBWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("imdb_title"),
		Name:              "sync-imdb",
		Interval:          24 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_imdb = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_imdb = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncDMMHashlistWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("dmm_hashlist"),
		Name:              "sync-dmm-hashlist",
		Interval:          6 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}
			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_dmm_hashlist = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_dmm_hashlist = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitMapIMDBTorrentWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("imdb_title"),
		Name:              "map-imdb-torrent",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 30 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}
			if running_worker.sync_bitmagnet {
				return true, "sync_bitmagnet is running"
			}
			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}
			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.map_imdb_torrent = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.map_imdb_torrent = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitMagnetCachePullerWorker(&WorkerConfig{
		Disabled: worker_queue.MagnetCachePullerQueue.Disabled,
		Name:     "pull-magnet-cache",
		Interval: 5 * time.Minute,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitMapAnimeIdWorker(&WorkerConfig{
		Disabled:     worker_queue.AnimeIdMapperQueue.Disabled,
		Name:         "map-anime-id",
		Interval:     10 * time.Minute,
		RunExclusive: true,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncAnimeAPIWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "sync-animeapi",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 45 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}

			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_animeapi = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_animeapi = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncAniDBTitlesWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "sync-anidb-titles",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_anidb_titles = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_anidb_titles = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncAniDBTVDBEpisodeMapWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "sync-anidb-tvdb-episode-map",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 45 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_anidb_titles {
				return true, "sync_anidb_titles is running"
			}

			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_anidb_tvdb_episode_map = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_anidb_tvdb_episode_map = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncManamiAnimeDatabaseWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "manami-anime-database",
		Interval:          6 * 24 * time.Hour,
		RunAtStartupAfter: 60 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_anidb_titles {
				return true, "sync_anidb_titles is running"
			}

			if running_worker.sync_animeapi {
				return true, "sync_animeapi is running"
			}

			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_manami_anime_database = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_manami_anime_database = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitMapAniDBTorrentWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "map-anidb-torrent",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 3 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}

			if running_worker.sync_anidb_titles {
				return true, "sync_anidb_titles is running"
			}

			if running_worker.sync_anidb_tvdb_episode_map {
				return true, "sync_anidb_tvdb_episode_map is running"
			}

			if running_worker.sync_animeapi {
				return true, "sync_animeapi is running"
			}

			if running_worker.sync_manami_anime_database {
				return true, "sync_manami_anime_database is running"
			}

			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncLetterboxdList(&WorkerConfig{
		Disabled:     worker_queue.LetterboxdListSyncerQueue.Disabled,
		Interval:     5 * time.Minute,
		Name:         "sync-letterboxd-list",
		OnEnd:        func() {},
		OnStart:      func() {},
		RunExclusive: true,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncBitmagnetWorker(&WorkerConfig{
		Disabled:          !config.Integration.Bitmagnet.IsEnabled(),
		Name:              "sync-bitmagnet",
		Interval:          60 * time.Minute,
		RunAtStartupAfter: 90 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}

			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}

			return false, ""
		},
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_bitmagnet = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_bitmagnet = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	return func() {
		for _, worker := range workers {
			worker.scheduler.Stop()
		}
	}
}
