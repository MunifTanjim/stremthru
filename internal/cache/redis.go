package cache

import (
	"context"
	"time"

	"github.com/MunifTanjim/stremthru/internal/redis"
	"github.com/elastic/go-freelru"
	rc "github.com/go-redis/cache/v9"
)

var (
	_ Cache[any] = (*RedisCache[any])(nil)
)

type localCache struct {
	c *freelru.LRU[string, []byte]
}

func (lc localCache) Set(key string, value []byte) {
	lc.c.Add(key, value)
}

func (lc localCache) Get(key string) ([]byte, bool) {
	return lc.c.Get(key)
}

func (lc localCache) Del(key string) {
	lc.c.Remove(key)
}

func newLocalCache(capacity uint32, lifetime time.Duration) localCache {
	lru, err := freelru.New[string, []byte](capacity, CacheHashKeyString)
	if err != nil {
		panic(err)
	}
	lru.SetLifetime(lifetime)
	return localCache{c: lru}
}

type RedisCache[V any] struct {
	c        *rc.Cache
	name     string
	lifetime time.Duration
}

func (cache *RedisCache[V]) GetName() string {
	return cache.name
}

func (cache *RedisCache[V]) Has(key string) bool {
	return cache.c.Exists(context.Background(), cache.name+":"+key)
}

func (cache *RedisCache[V]) Add(key string, value V) error {
	err := cache.c.Set(&rc.Item{
		Key:   cache.name + ":" + key,
		Value: value,
		TTL:   cache.lifetime,
	})
	return err
}

func (cache *RedisCache[V]) AddWithLifetime(key string, value V, lifetime time.Duration) error {
	err := cache.c.Set(&rc.Item{
		Key:   cache.name + ":" + key,
		Value: value,
		TTL:   lifetime,
	})
	return err
}

func (cache *RedisCache[V]) Get(key string, value *V) bool {
	err := cache.c.Get(context.Background(), cache.name+":"+key, value)
	if err != nil {
		return false
	}
	return true
}

func (cache *RedisCache[V]) Remove(key string) {
	cache.c.Delete(context.Background(), cache.name+":"+key)
}

func newRedisCache[V any](conf *CacheConfig) *RedisCache[V] {
	redisClient := redis.GetClient()
	if redisClient == nil {
		errMsg := "failed to create cache"
		if conf.Name != "" {
			errMsg += ": " + conf.Name
		}
		panic(errMsg)
	}

	if conf.Lifetime == 0 {
		conf.Lifetime = 5 * time.Minute
	}

	cache := &RedisCache[V]{
		c: rc.New(&rc.Options{
			Redis: redisClient,
			// LocalCache: newLocalCache(1024, conf.Lifetime/2),
		}),
		name:     conf.Name,
		lifetime: conf.Lifetime,
	}

	return cache
}
