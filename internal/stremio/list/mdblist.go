package stremio_list

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
)

var mdblistClient = mdblist.NewAPIClient(&mdblist.APIClientConfig{})

var mdblistLimitsCache = cache.NewCache[int](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "mdblist:limits",
})
