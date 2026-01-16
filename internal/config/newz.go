package config

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

var NewzNZBCacheSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_NZB_CACHE_SIZE"))
var NewzNZBCacheTTL = mustParseDuration("newz nzb cache ttl", getEnv("STREMTHRU_NEWZ_NZB_CACHE_TTL"), 6*time.Hour)
