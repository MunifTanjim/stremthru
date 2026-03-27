package dash_api

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var imdbPattern = regexp.MustCompile(`^tt\d+$`)
var hexPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9A-F]{40})$`)

type TorrentInfoItemResponse struct {
	Hash         string `json:"hash"`
	TorrentTitle string `json:"t_title"`
	Source       string `json:"src"`
	Category     string `json:"category"`
	Size         int64  `json:"size"`
	Indexer      string `json:"indexer"`
	Seeders      int    `json:"seeders"`
	Leechers     int    `json:"leechers"`
	Private      bool   `json:"private"`
	CreatedAt    string `json:"created_at"`
	IMDBID       string `json:"imdb_id"`
}

type ListTorrentInfoResponse struct {
	Items      []TorrentInfoItemResponse `json:"items"`
	NextCursor string                    `json:"next_cursor"`
}

func toTorrentInfoResponse(items []torrent_info.SearchItemsItem) []TorrentInfoItemResponse {
	responseItems := make([]TorrentInfoItemResponse, len(items))
	for i, item := range items {
		responseItems[i] = TorrentInfoItemResponse{
			Hash:         item.Hash,
			TorrentTitle: item.TorrentTitle,
			Source:       string(item.Source),
			Category:     string(item.Category),
			Size:         item.Size,
			Indexer:      item.Indexer,
			Seeders:      item.Seeders,
			Leechers:     item.Leechers,
			Private:      item.Private,
			CreatedAt:    item.CreatedAt.Time.Format(time.RFC3339),
			IMDBID:       item.IMDBId,
		}
	}
	return responseItems
}

func handleGetTorrentInfos(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query()
	q := query.Get("q")

	if q == "" {
		SendData(w, r, 200, ListTorrentInfoResponse{
			Items: []TorrentInfoItemResponse{},
		})
		return
	}

	var items []torrent_info.SearchItemsItem
	var err error
	nextCursor := ""

	if imdbPattern.MatchString(q) {
		items, err = torrent_info.SearchItemsByIMDBID(q)
	} else if hexPattern.MatchString(q) {
		item, itemErr := torrent_info.SearchItemByHash(strings.ToLower(q))
		if itemErr != nil {
			SendError(w, r, itemErr)
			return
		}
		if item != nil {
			items = []torrent_info.SearchItemsItem{*item}
		} else {
			items = []torrent_info.SearchItemsItem{}
		}
	} else {
		limit := util.SafeParseInt(query.Get("limit"), 20)
		params := torrent_info.SearchItemsByTitleParams{
			Title:  q,
			Cursor: query.Get("cursor"),
			Limit:  limit,
		}
		items, err = torrent_info.SearchItemsByTitle(params)
		if err == nil && len(items) == limit {
			nextCursor = items[len(items)-1].Hash
		}
	}

	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, ListTorrentInfoResponse{
		Items:      toTorrentInfoResponse(items),
		NextCursor: nextCursor,
	})
}

func AddTorrentInfoEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/torrents/infos", authed(handleGetTorrentInfos))
}
