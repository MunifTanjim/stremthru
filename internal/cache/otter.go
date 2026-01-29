package cache

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/maypok86/otter/v2"
)

var (
	_ Cache[any] = (*otterCache[any])(nil)
)

var persistentCaches []persistentCache
var persistentCachesMu sync.Mutex

type persistentCache interface {
	load()
	persist() error
}

func persistCaches() {
	persistentCachesMu.Lock()
	defer persistentCachesMu.Unlock()

	for _, c := range persistentCaches {
		c.persist()
	}
}

func registerPersistentCache(c persistentCache) {
	persistentCachesMu.Lock()
	defer persistentCachesMu.Unlock()
	persistentCaches = append(persistentCaches, c)
}

func ClosePersistentCaches() {
	persistCaches()
}

var cacheDir = filepath.Join(config.DataDir, "cache")

func init() {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			persistCaches()
		}
	}()
}

type otterCache[V any] struct {
	name     string
	lifetime time.Duration
	filePath string
	otter    *otter.Cache[string, V]
}

func (c *otterCache[V]) load() {
	if _, err := os.Stat(c.filePath); err == nil {
		otter.LoadCacheFromFile(c.otter, c.filePath)
	}
}

func (c *otterCache[V]) persist() error {
	return otter.SaveCacheToFile(c.otter, c.filePath)
}

func (c *otterCache[V]) GetName() string {
	return c.name
}

func (c *otterCache[V]) Add(key string, value V) error {
	return c.AddWithLifetime(key, value, c.lifetime)
}

func (c *otterCache[V]) AddWithLifetime(key string, value V, lifetime time.Duration) error {
	c.otter.Set(key, value)
	if lifetime > 0 {
		c.otter.SetExpiresAfter(key, lifetime)
	}
	return nil
}

func (c *otterCache[V]) Get(key string, value *V) bool {
	v, found := c.otter.GetIfPresent(key)
	if !found {
		return false
	}
	*value = v
	return true
}

func (c *otterCache[V]) Has(key string) bool {
	_, found := c.otter.GetIfPresent(key)
	return found
}

func (c *otterCache[V]) Remove(key string) {
	c.otter.Invalidate(key)
}

type cacheSizer interface {
	CacheSize() int64
}

func newOtterCache[V any](conf *CacheConfig) Cache[V] {
	if conf.DiskBacked {
		return newDiskBackedCache[V](conf)
	}

	filePath := filepath.Join(cacheDir, conf.Name+".gob")

	opts := &otter.Options[string, V]{}

	if conf.MaxSize > 0 {
		opts.MaximumWeight = uint64(conf.MaxSize)
		opts.Weigher = func(key string, value V) uint32 {
			if sizer, ok := any(value).(cacheSizer); ok {
				return uint32(sizer.CacheSize())
			}
			return 1
		}
	}

	cache := &otterCache[V]{
		name:     conf.Name,
		lifetime: conf.Lifetime,
		filePath: filePath,
		otter:    otter.Must(opts),
	}

	if conf.Persist {
		cache.load()
		registerPersistentCache(cache)
	}

	return cache
}
