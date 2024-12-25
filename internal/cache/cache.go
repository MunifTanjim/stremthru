package cache

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type Cache[V any] interface {
	GetName() string
	Add(key string, value V) error
	AddWithLifetime(key string, value V, lifetime time.Duration) error
	Get(key string, value *V) bool
	Remove(key string)
}

type CacheConfig struct {
	Lifetime time.Duration
	Name     string
}

func NewCache[V any](conf *CacheConfig) Cache[V] {
	if config.RedisURI != "" {
		return newRedisCache[V](conf)
	}

	return newLRUCache[V](conf)
}
