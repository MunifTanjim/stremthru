package cache

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/maypok86/otter/v2"
)

var (
	_ Cache[any] = (*diskBackedCache[any])(nil)
)

type diskBackedCacheMeta struct {
	Size int64
}

type diskBackedCache[V any] struct {
	name     string
	lifetime time.Duration
	dir      string
	filePath string
	otter    *otter.Cache[string, diskBackedCacheMeta]
}

func newDiskBackedCache[V any](conf *CacheConfig) Cache[V] {
	if conf.MaxSize <= 0 {
		conf.MaxSize = 1024 * 1024 * 1024 // 1 GB
	}

	dir := filepath.Join(cacheDir, conf.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}

	opts := &otter.Options[string, diskBackedCacheMeta]{
		OnAtomicDeletion: func(e otter.DeletionEvent[string, diskBackedCacheMeta]) {
			os.RemoveAll(filepath.Join(dir, e.Key))
		},
	}

	opts.MaximumWeight = uint64(conf.MaxSize)
	opts.Weigher = func(key string, val diskBackedCacheMeta) uint32 {
		return uint32(val.Size)
	}

	otterCache := otter.Must(opts)

	cache := &diskBackedCache[V]{
		name:     conf.Name,
		lifetime: conf.Lifetime,
		dir:      dir,
		filePath: filepath.Join(cacheDir, conf.Name+".gob"),
		otter:    otterCache,
	}

	cache.load()
	registerPersistentCache(cache)

	return cache
}

func (c *diskBackedCache[V]) load() {
	if _, err := os.Stat(c.filePath); err == nil {
		otter.LoadCacheFromFile(c.otter, c.filePath)
	}
	c.cleanOrphaned()
}

func (c *diskBackedCache[V]) cleanOrphaned() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		key := entry.Name()
		if _, found := c.otter.GetEntryQuietly(key); !found {
			os.Remove(c.getFilePath(key))
		}
	}

	for key := range c.otter.Keys() {
		if exists, _ := util.FileExists(c.getFilePath(key)); !exists {
			c.otter.Invalidate(key)
		}
	}
}

func (c *diskBackedCache[V]) persist() error {
	return otter.SaveCacheToFile(c.otter, c.filePath)
}

func (c *diskBackedCache[V]) GetName() string {
	return c.name
}

func (c *diskBackedCache[V]) getFilePath(key string) string {
	return filepath.Join(c.dir, key)
}

func (c *diskBackedCache[V]) Add(key string, value V) error {
	return c.AddWithLifetime(key, value, c.lifetime)
}

func (c *diskBackedCache[V]) AddWithLifetime(key string, value V, lifetime time.Duration) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}

	if err := os.WriteFile(c.getFilePath(key), buf.Bytes(), 0644); err != nil {
		return err
	}

	size := int64(1)
	if sizer, ok := any(value).(cacheSizer); ok {
		size = sizer.CacheSize()
	}
	c.otter.Set(key, diskBackedCacheMeta{Size: size})
	if lifetime > 0 {
		c.otter.SetExpiresAfter(key, lifetime)
	}
	return nil
}

func (c *diskBackedCache[V]) Get(key string, value *V) bool {
	data, err := os.ReadFile(c.getFilePath(key))
	if err != nil {
		c.otter.Invalidate(key)
		return false
	}
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(value); err != nil {
		return false
	}
	return true
}

func (c *diskBackedCache[V]) Has(key string) bool {
	_, found := c.otter.GetIfPresent(key)
	return found
}

func (c *diskBackedCache[V]) Remove(key string) {
	c.otter.Invalidate(key)
}
