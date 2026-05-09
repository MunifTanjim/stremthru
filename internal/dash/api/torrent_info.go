package dash_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

// IMDB Mapping Response Types
type IMDBMappingItemResponse struct {
	Hash      string `json:"hash"`
	TTitle    string `json:"t_title"`
	IMDBID    string `json:"imdb_id,omitempty"`
	IMDBTitle string `json:"imdb_title,omitempty"`
	IMDBYear  int    `json:"imdb_year,omitempty"`
	IMDBType  string `json:"imdb_type,omitempty"`
	MappedAt  string `json:"mapped_at"`
}

type ListIMDBMappingsResponse struct {
	Items      []IMDBMappingItemResponse `json:"items"`
	NextCursor string                    `json:"next_cursor"`
}

// AniDB Mapping Response Types
type AniDBMappingItemResponse struct {
	Hash       string `json:"hash"`
	TTitle     string `json:"t_title"`
	AniDBID    string `json:"anidb_id,omitempty"`
	AniDBTitle string `json:"anidb_title,omitempty"`
	SeasonType string `json:"season_type,omitempty"`
	Season     int    `json:"season,omitempty"`
	EpStart    int    `json:"ep_start,omitempty"`
	EpEnd      int    `json:"ep_end,omitempty"`
	MappedAt   string `json:"mapped_at"`
}

type ListAniDBMappingsResponse struct {
	Items      []AniDBMappingItemResponse `json:"items"`
	NextCursor string                     `json:"next_cursor"`
}

func toIMDBMappingResponse(items []imdb_torrent.MappingItem) []IMDBMappingItemResponse {
	responseItems := make([]IMDBMappingItemResponse, len(items))
	for i, item := range items {
		responseItems[i] = IMDBMappingItemResponse{
			Hash:      item.Hash,
			TTitle:    item.TTitle,
			IMDBID:    item.TId,
			IMDBTitle: item.IMDBTitle,
			IMDBYear:  item.IMDBYear,
			IMDBType:  item.IMDBType,
			MappedAt:  item.UAt.Time.Format(time.RFC3339),
		}
	}
	return responseItems
}

func toAniDBMappingResponse(items []anidb.MappingItem) []AniDBMappingItemResponse {
	responseItems := make([]AniDBMappingItemResponse, len(items))
	for i, item := range items {
		responseItems[i] = AniDBMappingItemResponse{
			Hash:       item.Hash,
			TTitle:     item.TTitle,
			AniDBID:    item.TId,
			AniDBTitle: item.AniDBTitle,
			SeasonType: string(item.SeasonType),
			Season:     item.Season,
			EpStart:    item.EpStart,
			EpEnd:      item.EpEnd,
			MappedAt:   item.UAt.Time.Format(time.RFC3339),
		}
	}
	return responseItems
}

func handleGetIMDBMappings(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query()
	mode := query.Get("mode")
	q := query.Get("q")
	cursor := query.Get("cursor")
	limit := util.SafeParseInt(query.Get("limit"), 100)
	unmapped := query.Get("unmapped") == "true"

	if q == "" {
		SendData(w, r, 200, ListIMDBMappingsResponse{
			Items: []IMDBMappingItemResponse{},
		})
		return
	}

	var items []imdb_torrent.MappingItem
	var err error

	switch mode {
	case "by-id":
		// q is a stream ID: tt1234567 or tt1234567:1:5
		if !strings.HasPrefix(q, "tt") {
			SendData(w, r, 200, ListIMDBMappingsResponse{
				Items: []IMDBMappingItemResponse{},
			})
			return
		}

		hashes, hashErr := torrent_info.ListHashesByStremId(q)
		if hashErr != nil {
			SendError(w, r, hashErr)
			return
		}

		items, err = imdb_torrent.GetMappingsByHashes(hashes, cursor, limit, false, false)
	case "by-title":
		hashes, hashErr := torrent_info.ListHashesByTitleQuery(q, 1000)
		if hashErr != nil {
			SendError(w, r, hashErr)
			return
		}
		items, err = imdb_torrent.GetMappingsByHashes(hashes, cursor, limit, !unmapped, unmapped)
	default:
		SendData(w, r, 200, ListIMDBMappingsResponse{
			Items: []IMDBMappingItemResponse{},
		})
		return
	}

	if err != nil {
		SendError(w, r, err)
		return
	}

	nextCursor := ""
	if len(items) == limit {
		nextCursor = items[len(items)-1].Hash
	}

	SendData(w, r, 200, ListIMDBMappingsResponse{
		Items:      toIMDBMappingResponse(items),
		NextCursor: nextCursor,
	})
}

func handleGetAniDBMappings(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query()
	mode := query.Get("mode")
	q := query.Get("q")
	cursor := query.Get("cursor")
	limit := util.SafeParseInt(query.Get("limit"), 100)
	unmapped := query.Get("unmapped") == "true"

	if q == "" {
		SendData(w, r, 200, ListAniDBMappingsResponse{
			Items: []AniDBMappingItemResponse{},
		})
		return
	}

	var items []anidb.MappingItem
	var err error

	switch mode {
	case "by-id":
		// q is a stream ID: anidb:1234 or anidb:1234:5
		if !strings.HasPrefix(q, "anidb:") {
			SendData(w, r, 200, ListAniDBMappingsResponse{
				Items: []AniDBMappingItemResponse{},
			})
			return
		}

		hashes, hashErr := torrent_info.ListHashesByStremId(q)
		if hashErr != nil {
			SendError(w, r, hashErr)
			return
		}

		items, err = anidb.GetMappingsByHashes(hashes, cursor, limit, true, false)
	case "by-title":
		hashes, hashErr := torrent_info.ListHashesByTitleQuery(q, 1000)
		if hashErr != nil {
			SendError(w, r, hashErr)
			return
		}
		items, err = anidb.GetMappingsByHashes(hashes, cursor, limit, !unmapped, unmapped)
	default:
		SendData(w, r, 200, ListAniDBMappingsResponse{
			Items: []AniDBMappingItemResponse{},
		})
		return
	}

	if err != nil {
		SendError(w, r, err)
		return
	}

	nextCursor := ""
	if len(items) == limit {
		nextCursor = items[len(items)-1].Hash
	}

	SendData(w, r, 200, ListAniDBMappingsResponse{
		Items:      toAniDBMappingResponse(items),
		NextCursor: nextCursor,
	})
}

func AddTorrentInfoEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/torrents/info/imdb", authed(handleGetIMDBMappings))
	router.HandleFunc("/torrents/info/anidb", authed(handleGetAniDBMappings))
}
