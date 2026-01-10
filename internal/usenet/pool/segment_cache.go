package usenet_pool

import (
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/maypok86/otter/v2"
)

var cacheLog = logger.Scoped("usenet/pool/segment_cache")

type SegmentCache struct {
	cache *otter.Cache[string, SegmentData]
}

func NewSegmentCache(size int64) *SegmentCache {
	cache := otter.Must(&otter.Options[string, SegmentData]{
		MaximumWeight: uint64(size),
		Weigher: func(key string, value SegmentData) uint32 {
			return uint32(value.Size())
		},
		OnAtomicDeletion: func(e otter.DeletionEvent[string, SegmentData]) {
			cacheLog.Trace("cache - deleted", "message_id", e.Key, "cause", e.Cause)
		},
	})

	return &SegmentCache{
		cache: cache,
	}
}

func (c *SegmentCache) Get(messageId string) (SegmentData, bool) {
	data, ok := c.cache.GetIfPresent(messageId)
	return data, ok
}

func (c *SegmentCache) Set(messageId string, data SegmentData) {
	c.cache.Set(messageId, data)
}
