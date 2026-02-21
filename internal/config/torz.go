package config

import "github.com/MunifTanjim/stremthru/internal/util"

type torzConfig struct {
	TorrentFileMaxSize int64
}

var Torz = func() torzConfig {
	torz := torzConfig{
		TorrentFileMaxSize: util.ToBytes(getEnv("STREMTHRU_TORZ_TORRENT_FILE_MAX_SIZE")),
	}

	return torz
}()
