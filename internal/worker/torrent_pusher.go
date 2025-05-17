package worker

import (
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	tss "github.com/MunifTanjim/stremthru/internal/torrent_stream/torrent_stream_syncinfo"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

var TorrentPusherQueue = IdQueue{
	debounceTime: 5 * time.Minute,
	transform: func(sid string) string {
		sid, _, _ = strings.Cut(sid, ":")
		return sid
	},
	disabled: !config.HasPeer || config.PeerAuthToken == "",
}

var Peer = peer.NewAPIClient(&peer.APIClientConfig{
	BaseURL: config.PeerURL,
	APIKey:  config.PeerAuthToken,
})

func InitPushTorrentsWorker(conf *WorkerConfig) *Worker {
	if TorrentPusherQueue.disabled {
		return nil
	}

	log := logger.Scoped("worker/torrent_pusher")

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(10 * time.Minute),
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

			TorrentPusherQueue.m.Range(func(k, v any) bool {
				sid, sidOk := k.(string)
				t, tOk := v.(time.Time)
				if sidOk && tOk && t.Before(time.Now()) {
					if tss.ShouldPush(sid) {
						if data, err := torrent_info.ListByStremId(sid, false); err == nil {
							params := &peer.PushTorrentsParams{
								Items: data.Items,
							}
							start := time.Now()
							if _, err := Peer.PushTorrents(params); err != nil {
								log.Error("failed to push torrents", "error", core.PackError(err), "duration", time.Since(start), "count", data.TotalItems)
							} else {
								log.Info("pushed torrents", "duration", time.Since(start), "count", data.TotalItems)
								tss.MarkPushed(sid)
							}
						} else {
							log.Error("failed to list torrents", "error", core.PackError(err), "sid", sid)
						}

					}

					TorrentPusherQueue.delete(sid)
				}
				return true
			})

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
