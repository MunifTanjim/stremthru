package worker

import (
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	tss "github.com/MunifTanjim/stremthru/internal/torrent_stream/torrent_stream_syncinfo"
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
	conf.Executor = func(w *Worker) error {
		log := w.Log

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
	}

	worker := NewWorker(conf)

	return worker
}
