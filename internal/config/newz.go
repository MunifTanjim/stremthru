package config

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

var NewzNZBCacheSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_NZB_CACHE_SIZE"))
var NewzNZBCacheTTL = mustParseDuration("newz nzb cache ttl", getEnv("STREMTHRU_NEWZ_NZB_CACHE_TTL"), 6*time.Hour)

var NewzSegmentCacheSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE"))
var NewzStreamBufferSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_STREAM_BUFFER_SIZE"))
var NewzMaxConnectionPerStream = util.MustParseInt(getEnv("STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM"))
