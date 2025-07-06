package worker

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/madflojo/tasks"
	"github.com/rs/xid"
)

var mutex sync.Mutex
var running_worker struct {
	sync_anidb_titles           bool
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
}

type WorkerConfig struct {
	Disabled          bool
	Executor          func(w *Worker) error
	JobTracker        *JobTracker[struct{}]
	JobIdTimeFormat   string
	Interval          time.Duration
	Log               *slog.Logger
	Name              string
	OnEnd             func()
	OnStart           func()
	RunAtStartupAfter time.Duration
	ShouldWait        func() (bool, string)
}

func NewWorker(conf *WorkerConfig) *Worker {
	if conf.Disabled {
		return nil
	}

	if conf.Log == nil {
		conf.Log = logger.Scoped("worker/" + conf.Name)
	}

	if conf.JobTracker != nil && conf.JobIdTimeFormat == "" {
		conf.JobIdTimeFormat = time.DateOnly + " 15"
	}

	log := conf.Log

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
		Log:        log,
	}

	jobId := ""
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          conf.Interval,
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else {
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

			shouldTrackJobId := conf.JobTracker != nil
			if shouldTrackJobId {
				jobId = time.Now().Format(conf.JobIdTimeFormat)
			} else {
				jobId = xid.New().String()
			}

			if shouldTrackJobId {
				job, err := conf.JobTracker.Get(jobId)
				if err != nil {
					return err
				}

				if job != nil && (job.Status == "done" || job.Status == "started") {
					log.Info("already done or started", "jobId", jobId, "status", job.Status)
					return nil
				}

				err = conf.JobTracker.Set(jobId, "started", "", nil)
				if err != nil {
					log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "started")
					return err
				}
			}

			if err = conf.Executor(worker); err != nil {
				return err
			}

			if shouldTrackJobId {
				err = conf.JobTracker.Set(jobId, "done", "", nil)
				if err != nil {
					log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "done")
					return err
				}
			}

			log.Info("done", "jobId", jobId)

			return err
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			if conf.JobTracker != nil {
				if terr := conf.JobTracker.Set(jobId, "failed", err.Error(), nil); terr != nil {
					log.Error("failed to set job status", "error", terr, "jobId", jobId, "status", "failed")
				}
			}

			jobId = ""
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
		Name:     "torrent_parser",
		Interval: 5 * time.Minute,
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

	if worker := InitPushTorrentsWorker(&WorkerConfig{
		Disabled: TorrentPusherQueue.disabled,
		Name:     "torrent_pusher",
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
		Name:     "store_crawler",
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
		Name:              "sync_imdb",
		Interval:          24 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		JobIdTimeFormat:   time.DateOnly,
		JobTracker: NewJobTracker("sync-imdb", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly, id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 24 * time.Hour))
		}),
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
		Name:              "sync_dmm_hashlist",
		Interval:          6 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		JobIdTimeFormat:   time.DateOnly + " 15",
		JobTracker: NewJobTracker("sync-dmm-hashlist", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly+" 15", id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 24 * time.Hour))
		}),
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
		Name:              "map_imdb_torrent",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 30 * time.Second,
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
		Name:     "magnet_cache_puller",
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
		Disabled: worker_queue.AnimeIdMapperQueue.Disabled,
		Name:     "map_anime_id",
		Interval: 5 * time.Minute,
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
		Name:              "sync_animeapi",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 45 * time.Second,
		JobIdTimeFormat:   time.DateOnly,
		JobTracker: NewJobTracker("sync-animeapi", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly, id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 24 * time.Hour))
		}),
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
		Name:              "sync_anidb_titles",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 30 * time.Second,
		JobIdTimeFormat:   time.DateOnly,
		JobTracker: NewJobTracker("sync-anidb-titles", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly, id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 24 * time.Hour))
		}),
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
		Name:              "sync_anidb_tvdb_episode_map",
		Interval:          1 * 24 * time.Hour,
		RunAtStartupAfter: 45 * time.Second,
		JobIdTimeFormat:   time.DateOnly,
		JobTracker: NewJobTracker("sync-anidb-tvdb-episode-map", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly, id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 24 * time.Hour))
		}),
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
		Name:              "sync_manami_anime_database",
		Interval:          6 * 24 * time.Hour,
		RunAtStartupAfter: 60 * time.Second,
		JobIdTimeFormat:   time.DateOnly,
		JobTracker: NewJobTracker("manami-anime-database", func(id string, job *Job[struct{}]) bool {
			date, err := time.Parse(time.DateOnly, id)
			if err != nil {
				return true
			}
			return date.Before(time.Now().Add(-7 * 6 * 24 * time.Hour))
		}),
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
		Name:              "map_anidb_torrent",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 90 * time.Second,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

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

	return func() {
		for _, worker := range workers {
			worker.scheduler.Stop()
		}
	}
}
