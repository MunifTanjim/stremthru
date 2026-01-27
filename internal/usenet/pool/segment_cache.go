package usenet_pool

import (
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/logger"
)

var cacheLog = logger.Scoped("usenet/pool/segment_cache")

type SegmentData struct {
	Body      []byte
	ByteRange ByteRange
	FileSize  int64
	Size      int64
}

type SegmentCache struct {
	cache *cache.OtterCache[SegmentData]
}

func NewSegmentCache(size int64) *SegmentCache {
	cache := cache.NewOtterCache[SegmentData](&cache.CacheConfig{
		Name:    "newz_segment",
		MaxSize: size,
		Persist: true,
	})

	return &SegmentCache{
		cache: cache,
	}
}

func (c *SegmentCache) Get(messageId string) (SegmentData, bool) {
	var data SegmentData
	ok := c.cache.Get(messageId, &data)
	return data, ok
}

func (c *SegmentCache) Set(messageId string, data SegmentData) {
	c.cache.Add(messageId, data)
}
