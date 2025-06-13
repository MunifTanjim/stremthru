package worker

import (
	"sync"

	"github.com/madflojo/tasks"
)

var mutex sync.Mutex
var running_worker struct {
	sync_dmm_hashlist bool
	sync_imdb         bool
	map_imdb_torrent  bool
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
		OnStart: func() {},
		OnEnd:   func() {},
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

	return func() {
		for _, worker := range workers {
			worker.scheduler.Stop()
		}
	}
}
