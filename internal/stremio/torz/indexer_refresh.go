package stremio_torz

import (
	"context"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/util"
)

// User-provided indexers are refreshed after responding with existing data.
// Bound the queue and deduplicate requests for the same user and media scope.
type indexerRefreshJob struct {
	key         string
	contentType string
	streamId    string
	ctx         *Ctx
}

var indexerRefreshJobs = make(chan indexerRefreshJob, 64)
var indexerRefreshLock sync.Mutex
var pendingIndexerRefresh = make(map[string]bool)
var recentIndexerRefresh = cache.NewLRUCache[bool](&cache.CacheConfig{
	Name:     "stremio:torz:indexer_refresh",
	Lifetime: 2 * time.Minute,
	MaxSize:  4096,
})

// An indexer's download URL may contain private tracker credentials. Store it
// temporarily per user configuration rather than putting it into torrent_info.
var indexerSourceLinks = cache.NewLRUCache[string](&cache.CacheConfig{
	Name:     "stremio:torz:indexer_source_links",
	Lifetime: time.Hour,
	MaxSize:  8192,
})

func sourceLinkCacheKey(userScope, hash string) string {
	return util.MD5Hash(userScope + "|" + hash)
}

func rememberIndexerSourceLink(userScope, hash, link string) {
	if userScope == "" || hash == "" || link == "" {
		return
	}
	_ = indexerSourceLinks.Add(sourceLinkCacheKey(userScope, hash), link)
}

func getIndexerSourceLink(userScope, hash string) string {
	if userScope == "" || hash == "" {
		return ""
	}
	var link string
	_ = indexerSourceLinks.Get(sourceLinkCacheKey(userScope, hash), &link)
	return link
}

func init() {
	if !config.Feature.HasPerformance() {
		return
	}
	for i := 0; i < 3; i++ {
		go func() {
			for job := range indexerRefreshJobs {
				func() {
					defer func() {
						_ = recentIndexerRefresh.Add(job.key, true)
						indexerRefreshLock.Lock()
						delete(pendingIndexerRefresh, job.key)
						indexerRefreshLock.Unlock()
					}()
					timeout := max(45*time.Second, config.Stremio.Torz.IndexerMaxTimeout)
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					_, hashes, err := GetStreamsFromIndexers(ctx, job.ctx, job.contentType, job.streamId)
					if err != nil {
						job.ctx.Log.Warn("background Torz indexer refresh failed", "error", err, "sid", job.streamId)
					} else {
						job.ctx.Log.Debug("background Torz indexer refresh completed", "sid", job.streamId, "count", len(hashes))
					}
				}()
			}
		}()
	}
}

// queueIndexerRefresh never delays a stream response or queues unbounded work.
func queueIndexerRefresh(ctx *Ctx, contentType, streamId, userScope string) {
	if !config.Feature.HasPerformance() {
		return
	}
	if len(ctx.Indexers) == 0 {
		return
	}
	key := util.MD5Hash(userScope + "|" + contentType + "|" + streamId)
	var recent bool
	if recentIndexerRefresh.Get(key, &recent) && recent {
		return
	}
	indexerRefreshLock.Lock()
	if pendingIndexerRefresh[key] {
		indexerRefreshLock.Unlock()
		return
	}
	pendingIndexerRefresh[key] = true
	select {
	case indexerRefreshJobs <- indexerRefreshJob{key: key, contentType: contentType, streamId: streamId, ctx: ctx}:
	default:
		delete(pendingIndexerRefresh, key)
		ctx.Log.Warn("background Torz indexer refresh queue is full", "sid", streamId)
	}
	indexerRefreshLock.Unlock()
}
