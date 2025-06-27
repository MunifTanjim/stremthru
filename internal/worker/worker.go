package worker

import (
	"sync"

	"github.com/madflojo/tasks"
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
}

type WorkerConfig struct {
	ShouldWait func() (bool, string)
	OnStart    func()
	OnEnd      func()
}

func InitWorkers() func() {
	workers := []*Worker{}

	if worker := InitParseTorrentWorker(&WorkerConfig{
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
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitCrawlStoreWorker(&WorkerConfig{
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
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitMapAnimeIdWorker(&WorkerConfig{
		ShouldWait: func() (bool, string) {
			return false, ""
		},
		OnStart: func() {},
		OnEnd:   func() {},
	}); worker != nil {
		workers = append(workers, worker)
	}

	if worker := InitSyncAnimeAPIWorker(&WorkerConfig{
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
