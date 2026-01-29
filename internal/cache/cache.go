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
	Has(key string) bool
	Remove(key string)
}

type CacheConfig struct {
	DiskBacked bool
	Lifetime   time.Duration
	MaxSize    int64
	Name       string
	Persist    bool
}

func NewCache[V any](conf *CacheConfig) Cache[V] {
	if conf.DiskBacked || conf.Persist {
		return newOtterCache[V](conf)
	}

	var v V
	if _, ok := any(v).(cacheSizer); ok {
		return newOtterCache[V](conf)
	}

	if conf.MaxSize == 0 {
		conf.MaxSize = 1024
	}

	if redis.IsAvailable() {
		return newRedisCache[V](conf)
	}

	return NewLRUCache[V](conf)
}
