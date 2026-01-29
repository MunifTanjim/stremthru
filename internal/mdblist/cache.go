package mdblist

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
)

var listCache = cache.NewCache[MDBListList](&cache.CacheConfig{
	Lifetime: 6 * time.Hour,
	Name:     "mdblist:list:v2",
	MaxSize:  1024,
})

var listIdByNameCache = cache.NewCache[string](&cache.CacheConfig{
	Lifetime: 12 * time.Hour,
	Name:     "mdblist:list-id-by-name:v2",
	MaxSize:  2048,
})
