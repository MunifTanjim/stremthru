package usenet_pool

import (
	"github.com/MunifTanjim/stremthru/internal/cache"
)

type SegmentCache struct {
	lru *cache.LRUCache[decodedData]
}

func NewSegmentCache(capacity int) *SegmentCache {
	if capacity <= 0 {
		return nil
	}

	lru := cache.NewLRUCache[decodedData](&cache.CacheConfig{
		Name:          "usenet/segment",
		LocalCapacity: uint32(capacity),
	})

	return &SegmentCache{
		lru: lru,
	}
}

func (c *SegmentCache) Get(messageId string) (decodedData, bool) {
	var data decodedData
	ok := c.lru.Get(messageId, &data)
	return data, ok
}

func (c *SegmentCache) Add(messageId string, data decodedData) {
	c.lru.Add(messageId, data)
}
