package dash_api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/letterboxd"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/tmdb"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
)

var cachedTorrentsStats = cache.NewCachedValue(cache.CachedValueConfig[*torrent_info.Stats]{
	Get: torrent_info.GetStats,
	TTL: 6 * time.Hour,
})

type CacheStatsEntry struct {
	Skipped int64 `json:"skipped"`
	Allowed int64 `json:"allowed"`
}

type TorrentsStats struct {
	TotalCount int `json:"total_count"`
	Files      struct {
		TotalCount int `json:"total_count"`
	} `json:"files"`
	Cache struct {
		TorrentInfo   CacheStatsEntry `json:"torrent_info"`
		TorrentStream CacheStatsEntry `json:"torrent_stream"`
	} `json:"cache"`
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

	tiSkipped, tiAllowed := torrent_info.GetUpsertCacheStats()
	tsSkipped, tsAllowed := torrent_stream.GetRecordCacheStats()
	data.Cache.TorrentInfo = CacheStatsEntry{Skipped: tiSkipped, Allowed: tiAllowed}
	data.Cache.TorrentStream = CacheStatsEntry{Skipped: tsSkipped, Allowed: tsAllowed}

	SendData(w, r, 200, data)
}

type IMDBTitleStats struct {
	TotalCount int `json:"total_count"`
}

var cachedIMDBTitleStats = cache.NewCachedValue(cache.CachedValueConfig[*IMDBTitleStats]{
	Get: func() (*IMDBTitleStats, error) {
		var count int
		err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s`, imdb_title.TableName)).Scan(&count)
		if err != nil {
			return nil, err
		}
		return &IMDBTitleStats{
			TotalCount: count,
		}, nil
	},
	TTL: 6 * time.Hour,
})

func HandleGetIMDBTitleStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	stats, err := cachedIMDBTitleStats.Get()
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, stats)
}

type ServerStatsFeature struct {
	Vault bool `json:"vault"`
}

type ServerStatsIntegration struct {
	Trakt bool `json:"trakt"`
}

type ServerStats struct {
	Version     string                 `json:"version"`
	StartedAt   time.Time              `json:"started_at"`
	Feature     ServerStatsFeature     `json:"feature"`
	Integration ServerStatsIntegration `json:"integration"`
}

func HandleGetServerStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	data := ServerStats{
		Version:   config.Version,
		StartedAt: config.ServerStartTime,
		Feature: ServerStatsFeature{
			Vault: config.Feature.HasVault(),
		},
		Integration: ServerStatsIntegration{
			Trakt: config.Integration.Trakt.IsEnabled(),
		},
	}
	SendData(w, r, 200, data)
}

type ListsStats struct {
	AniList struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"anilist"`
	Letterboxd struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"letterboxd"`
	MDBList struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"mdblist"`
	TMDB struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"tmdb"`
	Trakt struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"trakt"`
	TVDB struct {
		TotalLists int `json:"total_lists"`
		TotalItems int `json:"total_items"`
	} `json:"tvdb"`
}

var query_get_lists_stats = fmt.Sprintf(`
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
UNION ALL
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
UNION ALL
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
UNION ALL
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
UNION ALL
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
UNION ALL
SELECT COUNT(1) FROM %s UNION ALL SELECT COUNT(1) FROM %s
`,
	anilist.ListTableName, anilist.MediaTableName,
	letterboxd.ListTableName, letterboxd.ItemTableName,
	mdblist.ListTableName, mdblist.ItemTableName,
	tmdb.ListTableName, tmdb.ItemTableName,
	trakt.ListTableName, trakt.ItemTableName,
	tvdb.ListTableName, tvdb.ItemTableName,
)

var cachedListsStats = cache.NewCachedValue(cache.CachedValueConfig[*ListsStats]{
	Get: func() (*ListsStats, error) {
		rows, err := db.Query(query_get_lists_stats)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		counts := make([]int, 0, 12)
		for rows.Next() {
			var count int
			if err := rows.Scan(&count); err != nil {
				return nil, err
			}
			counts = append(counts, count)
		}

		stats := ListsStats{}
		stats.AniList.TotalLists = counts[0]
		stats.AniList.TotalItems = counts[1]
		stats.Letterboxd.TotalLists = counts[2]
		stats.Letterboxd.TotalItems = counts[3]
		stats.MDBList.TotalLists = counts[4]
		stats.MDBList.TotalItems = counts[5]
		stats.TMDB.TotalLists = counts[6]
		stats.TMDB.TotalItems = counts[7]
		stats.Trakt.TotalLists = counts[8]
		stats.Trakt.TotalItems = counts[9]
		stats.TVDB.TotalLists = counts[10]
		stats.TVDB.TotalItems = counts[11]

		return &stats, nil
	},
	TTL: 3 * time.Hour,
})

func HandleGetListsStats(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	stats, err := cachedListsStats.Get()
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, stats)
}
