package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
)

var cachedTorrentsStats = cache.NewCachedValue(cache.CachedValueConfig[*torrent_info.Stats]{
	Get: torrent_info.GetStats,
	TTL: 6 * time.Hour,
})

type TorrentsStats struct {
	TotalCount int `json:"total_count"`
	Files      struct {
		TotalCount int `json:"total_count"`
	} `json:"files"`
}

func HandleGetTorrentsStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	stats, err := cachedTorrentsStats.Get()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := TorrentsStats{}
	data.TotalCount = stats.TotalCount
	data.Files.TotalCount = stats.Streams.TotalCount
	SendData(w, r, 200, data)
}

type ServerStats struct {
	Version   string    `json:"version"`
	StartedAt time.Time `json:"started_at"`
}

func HandleGetServerStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	data := ServerStats{
		Version:   config.Version,
		StartedAt: config.ServerStartTime,
	}
	SendData(w, r, 200, data)
}
