package dash_api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/letterboxd"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/tmdb"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
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
