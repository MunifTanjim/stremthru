package dash_api

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	usenet_server "github.com/MunifTanjim/stremthru/internal/usenet/server"
	usenet_stats "github.com/MunifTanjim/stremthru/internal/usenet/stats"
)

func parseRangeSince(r *http.Request) (since time.Time, interval string) {
	now := time.Now()
	switch r.URL.Query().Get("range") {
	case "24h":
		return now.Add(-24 * time.Hour), "hour"
	case "7d":
		return now.AddDate(0, 0, -7), "day"
	case "30d":
		return now.AddDate(0, 0, -30), "day"
	default:
		return now.Add(-24 * time.Hour), "hour"
	}
}

func getUsenetServerNames() map[string]string {
	names := map[string]string{}
	if servers, err := usenet_server.GetAll(); err == nil {
		for _, s := range servers {
			names[s.Id] = s.Name
		}
	}
	return names
}

type UsenetServerStatsHistoryResponse struct {
	Items []usenet_stats.AggregatedServerStats `json:"items"`
}

func HandleGetUsenetServerStatsHistory(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	since, _ := parseRangeSince(r)

	servers, err := usenet_stats.GetAggregatedStats(since)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if servers == nil {
		servers = []usenet_stats.AggregatedServerStats{}
	}

	names := getUsenetServerNames()
	for i := range servers {
		servers[i].ServerName = names[servers[i].ServerId]
	}

	SendData(w, r, 200, UsenetServerStatsHistoryResponse{Items: servers})
}

type UsenetServerStatsTimeSeriesResponse struct {
	Items map[string]*usenet_stats.ServerTimeSeries `json:"items"`
}

func HandleGetUsenetServerStatsTimeSeries(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	since, interval := parseRangeSince(r)

	servers, err := usenet_stats.GetTimeSeriesStats(since, interval)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if servers == nil {
		servers = map[string]*usenet_stats.ServerTimeSeries{}
	}

	names := getUsenetServerNames()
	for id, s := range servers {
		s.Name = names[id]
	}

	SendData(w, r, 200, UsenetServerStatsTimeSeriesResponse{Items: servers})
}
