package config

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type torzConfig struct {
	TorrentFileCacheSize int64
	TorrentFileCacheTTL  time.Duration
	TorrentFileMaxSize   int64
}

var Torz = func() torzConfig {
	torz := torzConfig{
		TorrentFileCacheSize: util.ToBytes(getEnv("STREMTHRU_TORZ_TORRENT_FILE_CACHE_SIZE")),
		TorrentFileCacheTTL:  mustParseDuration("torz torrent file cache ttl", getEnv("STREMTHRU_TORZ_TORRENT_FILE_CACHE_TTL")),
		TorrentFileMaxSize:   util.ToBytes(getEnv("STREMTHRU_TORZ_TORRENT_FILE_MAX_SIZE")),
	}

	return torz
}()
