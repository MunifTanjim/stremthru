package worker

import (
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/job_log"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/madflojo/tasks"
)

var mutex sync.Mutex
var running_worker struct {
	sync_anidb_titles           bool
	sync_animetosho             bool
	sync_bitmagnet              bool
	sync_dmm_hashlist           bool
	sync_imdb                   bool
	map_anidb_torrent           bool
	map_imdb_torrent            bool
	sync_animeapi               bool
	sync_anidb_tvdb_episode_map bool
	sync_manami_anime_database  bool
}

type Worker struct {
	scheduler  *tasks.Scheduler
	shouldSkip func() bool
	shouldWait func() (bool, string)
	onStart    func()
	onEnd      func()
	Log        *logger.Logger
	jobTracker *JobTracker[struct{}]
}

type WorkerConfig struct {
	Disabled          bool
	Executor          func(w *Worker) error
	Interval          time.Duration
	HeartbeatInterval time.Duration
	Log               *logger.Logger
	Name              string
	OnEnd             func()
	OnStart           func()
	RunAtStartupAfter time.Duration
	RunExclusive      bool
	ShouldSkip        func() bool
	ShouldWait        func() (bool, string)
}

type WorkerDetail struct {
	Id       string        `json:"id"`
	Title    string        `json:"title"`
	Interval time.Duration `json:"interval"`
}

var WorkerDetailsById = map[string]*WorkerDetail{
	"parse-torrent": {
		Title: "Parse Torrent",
	},
	"push-torrent": {
		Title: "Push Torrent",
	},
	"crawl-store": {
		Title: "Crawl Store",
	},
	"sync-imdb": {
		Title: "Sync IMDB",
	},
	"sync-dmm-hashlist": {
		Title: "Sync DMM Hashlist",
	},
	"map-imdb-torrent": {
		Title: "Map IMDB Torrent",
	},
	"pull-magnet-cache": {
		Title: "Pull Magnet Cache",
	},
	"map-anime-id": {
		Title: "Map Anime ID",
	},
	"sync-animeapi": {
		Title: "Sync AnimeAPI",
	},
	"sync-anidb-titles": {
		Title: "Sync AniDB Titles",
	},
	"sync-anidb-tvdb-episode-map": {
		Title: "Sync AniDB-TVDB Episode Map",
	},
	"manami-anime-database": {
		Title: "Sync Manami Anime Database",
	},
	"map-anidb-torrent": {
		Title: "Map AniDB Torrent",
	},
	"sync-letterboxd-list": {
		Title: "Sync Letterboxd List",
	},
	"sync-bitmagnet": {
		Title: "Sync Bitmagnet",
	},
	"sync-animetosho": {
		Title: "Sync AnimeTosho",
	},
	"reload-linked-userdata-addon": {
		Title: "Reload Linked Userdata Addon",
	},
	"sync-stremio-trakt": {
		Title: "Sync Stremio-Trakt",
	},
	"sync-stremio-stremio": {
		Title: "Sync Stremio-Stremio",
	},
}

func NewWorker(conf *WorkerConfig) *Worker {
	if conf.Name == "" {
		panic("worker name cannot be empty")
	}

	if details, ok := WorkerDetailsById[conf.Name]; !ok {
		panic("worker details not present: " + conf.Name)
	} else {
		details.Id = conf.Name
		details.Interval = conf.Interval
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

	if conf.OnStart == nil {
		conf.OnStart = func() {}
	}
	if conf.OnEnd == nil {
		conf.OnEnd = func() {}
	}
	if conf.ShouldSkip == nil {
		conf.ShouldSkip = func() bool {
			return false
		}
	}
	if conf.ShouldWait == nil {
		conf.ShouldWait = func() (bool, string) {
			return false, ""
		}
	}

	log := conf.Log

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldSkip: conf.ShouldSkip,
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

			if worker.shouldSkip != nil && worker.shouldSkip() {
				log.Trace("skipping")
				return nil
			}

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

			var tjob *job_log.ParsedJobLog[struct{}]
			if conf.RunExclusive {
				tjob, err = jobTracker.GetLast()
				if err != nil {
					return err
				}
				if tjob != nil {
					status := tjob.Status
					switch status {
					case "started":
						if !util.HasDurationPassedSince(tjob.UpdatedAt, conf.HeartbeatInterval+heartbeatIntervalTolerance) {
							if util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
								log.Warn("skipping, last job is still running, for too long", "jobId", tjob.Id, "status", status)
							} else {
								log.Info("skipping, last job is still running", "jobId", tjob.Id, "status", status)
							}
							return nil
						}

						log.Warn("last job heartbeat timed out, restarting", "jobId", tjob.Id, "status", status)
						if err := jobTracker.Set(tjob.Id, "failed", "heartbeat timed out", nil); err != nil {
							log.Error("failed to set last job status", "error", err, "jobId", tjob.Id, "status", "failed")
						}
					case "done":
						if !util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
							log.Info("already done", "jobId", tjob.Id, "status", status)
							return nil
						}
					case "failed":
						log.Warn("last job failed", "jobId", tjob.Id, "status", status, "error", tjob.Error)
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
		Disabled:     !config.Feature.HasTorrentInfo(),
		Name:         "parse-torrent",
		Interval:     5 * time.Minute,
		RunExclusive: true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_animetosho {
				return true, "sync_animetosho is running"
			}
			if running_worker.sync_bitmagnet {
				return true, "sync_bitmagnet is running"
			}
			if running_worker.sync_dmm_hashlist {
				return true, "sync_dmm_hashlist is running"
			}
			if running_worker.sync_imdb {
				return true, "sync_imdb is running"
			}
			if running_worker.map_anidb_torrent {
				return true, "map_anidb_torrent is running"
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
		Disabled: worker_queue.StoreCrawlerQueue.Disabled,
		Name:     "crawl-store",
		Interval: 30 * time.Minute,
		ShouldSkip: func() bool {
			return worker_queue.StoreCrawlerQueue.IsEmpty()
		},
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
		Disabled:          !config.Feature.HasIMDBTitle(),
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
		Disabled:          !config.Feature.HasDMMHashlist(),
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
		Disabled:          !config.Feature.HasIMDBTitle(),
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
			if running_worker.sync_animetosho {
				return true, "sync_animetosho is running"
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
		ShouldSkip: func() bool {
			return worker_queue.MagnetCachePullerQueue.IsEmpty()
		},
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
		ShouldSkip: func() bool {
			return worker_queue.AnimeIdMapperQueue.IsEmpty()
		},
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
		RunAtStartupAfter: 90 * time.Second,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			mutex.Lock()
			defer mutex.Unlock()

			if running_worker.sync_animetosho {
				return true, "sync_animetosho is running"
			}
			if running_worker.sync_bitmagnet {
				return true, "sync_bitmagnet is running"
			}
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
		OnStart: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.map_anidb_torrent = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.map_anidb_torrent = false
		},
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
		ShouldSkip: func() bool {
			return worker_queue.LetterboxdListSyncerQueue.IsEmpty()
		},
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

	if worker := InitSyncAnimeToshoWorker(&WorkerConfig{
		Disabled:          !config.Feature.IsEnabled("anime"),
		Name:              "sync-animetosho",
		Interval:          24 * time.Hour,
		RunAtStartupAfter: 90 * time.Second,
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

			running_worker.sync_animetosho = true
		},
		OnEnd: func() {
			mutex.Lock()
			defer mutex.Unlock()

			running_worker.sync_animetosho = false
		},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitLinkedUserdataAddonReloaderWorker(&WorkerConfig{
		Disabled: worker_queue.LinkedUserdataAddonReloaderQueue.Disabled,
		Name:     "reload-linked-userdata-addon",
		Interval: 5 * time.Minute,
		ShouldSkip: func() bool {
			return worker_queue.LinkedUserdataAddonReloaderQueue.IsEmpty()
		},
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncStremioTraktWorker(&WorkerConfig{
		Disabled:          !config.Feature.HasVault() || !config.Integration.Trakt.IsEnabled(),
		Name:              "sync-stremio-trakt",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 5 * time.Minute,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncStremioStremioWorker(&WorkerConfig{
		Disabled:          !config.Feature.HasVault(),
		Name:              "sync-stremio-stremio",
		Interval:          30 * time.Minute,
		RunAtStartupAfter: 5 * time.Minute,
		RunExclusive:      true,
		ShouldWait: func() (bool, string) {
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
