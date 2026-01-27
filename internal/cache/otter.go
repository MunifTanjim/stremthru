package cache

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/maypok86/otter/v2"
)

var otterPersistentCaches []otterCacheSaver
var otterPersistentCachesMu sync.Mutex

type otterCacheSaver interface {
	save() error
}

func init() {
	cacheDir := filepath.Join(config.DataDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			persistOtterCaches()
		}
	}()
}

func persistOtterCaches() {
	otterPersistentCachesMu.Lock()
	defer otterPersistentCachesMu.Unlock()

	for _, c := range otterPersistentCaches {
		c.save()
	}
}

func CloseOtterCaches() {
	persistOtterCaches()
}

func registerOtterCache(c otterCacheSaver) {
	otterPersistentCachesMu.Lock()
	defer otterPersistentCachesMu.Unlock()
	otterPersistentCaches = append(otterPersistentCaches, c)
}

type OtterCache[V any] struct {
	name     string
	lifetime time.Duration
	filePath string
	otter    *otter.Cache[string, V]
}

func (c *OtterCache[V]) GetName() string {
	return c.name
}

func (c *OtterCache[V]) save() error {
	return otter.SaveCacheToFile(c.otter, c.filePath)
}

func (c *OtterCache[V]) Add(key string, value V) error {
	return c.AddWithLifetime(key, value, c.lifetime)
}

func (c *OtterCache[V]) AddWithLifetime(key string, value V, lifetime time.Duration) error {
	c.otter.Set(key, value)
	if lifetime > 0 {
		c.otter.SetExpiresAfter(key, lifetime)
	}
	return nil
}

func (c *OtterCache[V]) Get(key string, value *V) bool {
	v, found := c.otter.GetIfPresent(key)
	if !found {
		return false
	}
	*value = v
	return true
}

func (c *OtterCache[V]) Has(key string) bool {
	_, found := c.otter.GetIfPresent(key)
	return found
}

func (c *OtterCache[V]) Remove(key string) {
	c.otter.Invalidate(key)
}

func NewOtterCache[V any](conf *CacheConfig) *OtterCache[V] {
	cacheDir := filepath.Join(config.DataDir, "cache")
	filePath := filepath.Join(cacheDir, conf.Name+".gob")

	opts := &otter.Options[string, V]{}

	if conf.MaxSize > 0 {
		opts.MaximumWeight = uint64(conf.MaxSize)
		opts.Weigher = func(key string, value V) uint32 {
			if sizer, ok := any(value).(interface{ Size() int64 }); ok {
				return uint32(sizer.Size())
			}
			return 1
		}
	} else if conf.MaxCount > 0 {
		opts.MaximumSize = conf.MaxCount
	}

	otterCache := otter.Must(opts)

	if conf.Persist {
		if _, err := os.Stat(filePath); err == nil {
			otter.LoadCacheFromFile(otterCache, filePath)
		}
	}

	cache := &OtterCache[V]{
		name:     conf.Name,
		lifetime: conf.Lifetime,
		filePath: filePath,
		otter:    otterCache,
	}

	if conf.Persist {
		registerOtterCache(cache)
	}

	return cache
}
