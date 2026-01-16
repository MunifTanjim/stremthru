package usenet_pool

import (
	"sync"

	"github.com/MunifTanjim/stremthru/internal/cache"
)

type SegmentData struct {
	Body      []byte
	ByteRange ByteRange
	FileSize  int64
	Size      int64
}

func (sd *SegmentData) CacheSize() int64 {
	return sd.Size
}

type SegmentCache interface {
	Get(messageId string) (SegmentData, bool)
	Set(messageId string, data SegmentData)
}

var (
	_ SegmentCache = (*segmentCache)(nil)
	_ SegmentCache = (*noopSegmentCache)(nil)
)

type segmentCache struct {
	cache cache.Cache[SegmentData]
}

func NewSegmentCache(size int64) SegmentCache {
	cache := cache.NewCache[SegmentData](&cache.CacheConfig{
		Name:       "newz_segment",
		MaxSize:    size,
		DiskBacked: true,
	})

	return &segmentCache{
		cache: cache,
	}
}

func (c *segmentCache) Get(messageId string) (SegmentData, bool) {
	var data SegmentData
	ok := c.cache.Get(messageId, &data)
	return data, ok
}

func (c *segmentCache) Set(messageId string, data SegmentData) {
	c.cache.Add(messageId, data)
}

type noopSegmentCache struct{}

func (n *noopSegmentCache) Get(messageId string) (SegmentData, bool) {
	return SegmentData{}, false
}

func (n *noopSegmentCache) Set(messageId string, data SegmentData) {
}

var getNoopSegmentCache = sync.OnceValue(func() SegmentCache {
	return &noopSegmentCache{}
})
