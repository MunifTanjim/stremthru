package cache

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/elastic/go-freelru"
	rc "github.com/go-redis/cache/v9"
	r "github.com/redis/go-redis/v9"
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

type redisConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

func parseRedisConnectionURI(uri string) (*redisConfig, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	config := redisConfig{
		Addr:     u.Host,
		Username: u.User.Username(),
		Password: "",
		DB:       0,
	}
	password, _ := u.User.Password()
	config.Password = password
	if db, err := strconv.Atoi(strings.TrimPrefix(u.Path, "/")); err == nil {
		config.DB = db
	}
	return &config, nil
}

var redis = func() *r.Client {
	if config.RedisURI == "" {
		return nil
	}

	rconf, err := parseRedisConnectionURI(config.RedisURI)
	if err != nil {
		return nil
	}

	redis := r.NewClient(&r.Options{
		Addr:     rconf.Addr,
		Username: rconf.Username,
		Password: rconf.Password,
		DB:       rconf.DB,
	})

	return redis
}()

type RedisCache[V any] struct {
	c        *rc.Cache
	name     string
	lifetime time.Duration
}

func (cache *RedisCache[V]) GetName() string {
	return cache.name
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
	if redis == nil {
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
			Redis:      redis,
			LocalCache: newLocalCache(1024, conf.Lifetime/2),
		}),
		name:     conf.Name,
		lifetime: conf.Lifetime,
	}

	return cache
}
