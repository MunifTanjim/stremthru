package buddy

import (
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
)

// Keep peer refreshes away from the HTTP request path. In-flight deduplication
// and a short cooldown prevent a popular series from spawning many peer pulls.
var queuedPeerPulls sync.Map
var peerPullJobs = make(chan string, 128)
var recentPeerPulls = cache.NewLRUCache[bool](&cache.CacheConfig{
	Name:     "buddy:recent_peer_pulls",
	Lifetime: time.Minute,
	MaxSize:  4096,
})

func init() {
	if !config.Feature.HasPerformance() {
		return
	}
	for i := 0; i < 3; i++ {
		go func() {
			for sid := range peerPullJobs {
				func() {
					defer queuedPeerPulls.Delete(sid)
					PullTorrentsByStremId(sid, "")
					_ = recentPeerPulls.Add(sid, true)
				}()
			}
		}()
	}
}

// QueuePullTorrentsByStremId is deliberately non-blocking. Existing stored
// results are returned immediately; new peer results appear on the next read.
func QueuePullTorrentsByStremId(sid string) {
	if !config.Feature.HasPerformance() {
		return
	}
	sid = torrent_stream.CleanStremId(sid)
	if sid == "" {
		return
	}
	var recentlyPulled bool
	if recentPeerPulls.Get(sid, &recentlyPulled) && recentlyPulled {
		return
	}
	if _, loaded := queuedPeerPulls.LoadOrStore(sid, struct{}{}); loaded {
		return
	}
	select {
	case peerPullJobs <- sid:
	default:
		queuedPeerPulls.Delete(sid)
		pullPeerLog.Warn("peer pull queue is full; skipping refresh", "sid", sid)
	}
}
