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

func (sd *SegmentData) GetSize() int64 {
	return sd.Size
}

type SegmentCache struct {
	cache cache.Cache[SegmentData]
}

func NewSegmentCache(size int64) *SegmentCache {
	cache := cache.NewCache[SegmentData](&cache.CacheConfig{
		Name:       "newz_segment",
		MaxSize:    size,
		DiskBacked: true,
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
