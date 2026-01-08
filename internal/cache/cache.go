package cache

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/redis"
)

type Cache[V any] interface {
	GetName() string
	Add(key string, value V) error
	AddWithLifetime(key string, value V, lifetime time.Duration) error
	Get(key string, value *V) bool
	Remove(key string)
}

type CacheConfig struct {
	Lifetime      time.Duration
	Name          string
	LocalCapacity uint32
}

func NewCache[V any](conf *CacheConfig) Cache[V] {
	if conf.LocalCapacity == 0 {
		conf.LocalCapacity = 1024
	}

	if redis.IsAvailable() {
		return newRedisCache[V](conf)
	}

	return NewLRUCache[V](conf)
}
