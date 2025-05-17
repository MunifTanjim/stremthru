package worker

import (
	"slices"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/logger"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

func InitParseTorrentWorker(conf *WorkerConfig) *Worker {
	log := logger.Scoped("worker/torrent_parser")

	var parseTorrentInfo = func(t *ti.TorrentInfo) *ti.TorrentInfo {
		if t.ParserVersion > ptt.Version().Int() {
			return nil
		}

		err := t.ForceParse()
		if err != nil {
			log.Warn("failed to parse", "error", err, "title", t.TorrentTitle)
			return nil
		}

		return t
	}

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(5 * time.Minute),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
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

			for {
				tInfos, err := ti.GetUnparsed(5000)
				if err != nil {
					return err
				}

				for cTInfos := range slices.Chunk(tInfos, 500) {
					parsedTInfos := []*ti.TorrentInfo{}
					for i := range cTInfos {
						if t := parseTorrentInfo(&cTInfos[i]); t != nil {
							parsedTInfos = append(parsedTInfos, t)
						}
					}
					if err := ti.UpsertParsed(parsedTInfos); err != nil {
						return err
					}
					log.Info("upserted parsed torrent info", "count", len(parsedTInfos))
					time.Sleep(1 * time.Second)
				}

				if len(tInfos) < 5000 {
					break
				}

				time.Sleep(5 * time.Second)
			}

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	return worker
}
