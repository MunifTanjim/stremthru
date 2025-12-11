package trakt

import (
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type HistoryItemType string

const (
	HistoryItemTypeAll      HistoryItemType = ""
	HistoryItemTypeMovies   HistoryItemType = "movies"
	HistoryItemTypeShows    HistoryItemType = "shows"
	HistoryItemTypeSeasons  HistoryItemType = "seasons"
	HistoryItemTypeEpisodes HistoryItemType = "episodes"
)

type HistoryItemAction string

const (
	HistoryItemActionScrobble HistoryItemAction = "scrobble"
	HistoryItemActionCheckin  HistoryItemAction = "checkin"
	HistoryItemActionWatch    HistoryItemAction = "watch"
)

type HistoryItem struct {
	Id        int64               `json:"id"`
	WatchedAt time.Time           `json:"watched_at"`
	Action    HistoryItemAction   `json:"action"`
	Type      ItemType            `json:"type"` // "movie" or "episode"
	Movie     *ListItemMovie      `json:"movie,omitempty"`
	Episode   *MinimalItemEpisode `json:"episode,omitempty"`
	Show      *ListItemShow       `json:"show,omitempty"`
}

type GetHistoryData = []HistoryItem

type GetHistoryParams struct {
	Ctx
	Type    HistoryItemType
	Id      int
	StartAt *time.Time
	EndAt   *time.Time
	Page    int
	Limit   int
}

func (c APIClient) GetHistory(params *GetHistoryParams) (request.APIResponse[GetHistoryData], error) {
	path := "/sync/history"
	if params.Type != "" {
		path += "/" + string(params.Type)
		if params.Id != 0 {
			path += "/" + strconv.Itoa(params.Id)
		}
	}

	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		params.Query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.StartAt != nil {
		params.Query.Set("start_at", params.StartAt.UTC().Format(time.RFC3339))
	}
	if params.EndAt != nil {
		params.Query.Set("end_at", params.EndAt.UTC().Format(time.RFC3339))
	}

	response := paginatedResponseData[HistoryItem]{}
	res, err := c.Request("GET", path, params, &response)
	return request.NewAPIResponse(res, response.data), err
}

type SyncHistoryParamsItem struct {
	WatchedAt *time.Time  `json:"watched_at,omitempty"`
	Ids       ListItemIds `json:"ids"`
}

type SyncHistoryShow struct {
	SyncHistoryParamsItem
	Seasons []SyncHistoryShowSeason `json:"seasons,omitempty"`
}

type SyncHistoryShowSeason struct {
	WatchedAt *time.Time                     `json:"watched_at,omitempty"`
	Number    int                            `json:"number"`
	Episodes  []SyncHistoryShowSeasonEpisode `json:"episodes,omitempty"`
}

type SyncHistoryShowSeasonEpisode struct {
	WatchedAt *time.Time `json:"watched_at,omitempty"`
	Number    int        `json:"number"`
}

type SyncHistoryResponseNotFoundItem struct {
	Ids ListItemIds `json:"ids"`
}

type AddToHistoryData struct {
	ResponseError
	Added struct {
		Movies   int `json:"movies"`
		Episodes int `json:"episodes"`
	} `json:"added"`
	NotFound struct {
		Movies   []SyncHistoryResponseNotFoundItem `json:"movies"`
		Shows    []SyncHistoryResponseNotFoundItem `json:"shows"`
		Seasons  []SyncHistoryResponseNotFoundItem `json:"seasons"`
		Episodes []SyncHistoryResponseNotFoundItem `json:"episodes"`
		Ids      []int64                           `json:"ids"`
	} `json:"not_found"`
}

type AddToHistoryParams struct {
	Ctx
	Movies   []SyncHistoryParamsItem `json:"movies,omitempty"`
	Shows    []SyncHistoryShow       `json:"shows,omitempty"`
	Seasons  []SyncHistoryParamsItem `json:"seasons,omitempty"`
	Episodes []SyncHistoryParamsItem `json:"episodes,omitempty"`
}

func (c APIClient) AddToHistory(params *AddToHistoryParams) (request.APIResponse[AddToHistoryData], error) {
	params.JSON = params
	response := AddToHistoryData{}
	res, err := c.Request("POST", "/sync/history", params, &response)
	return request.NewAPIResponse(res, response), err
}

type RemoveFromHistoryData struct {
	ResponseError
	Deleted struct {
		Movies   int `json:"movies"`
		Episodes int `json:"episodes"`
	} `json:"deleted"`
	NotFound struct {
		Movies   []SyncHistoryResponseNotFoundItem `json:"movies"`
		Shows    []SyncHistoryResponseNotFoundItem `json:"shows"`
		Seasons  []SyncHistoryResponseNotFoundItem `json:"seasons"`
		Episodes []SyncHistoryResponseNotFoundItem `json:"episodes"`
		Ids      []int64                           `json:"ids"`
	} `json:"not_found"`
}

type RemoveFromHistoryParams struct {
	Ctx
	Movies   []SyncHistoryParamsItem `json:"movies,omitempty"`
	Shows    []SyncHistoryShow       `json:"shows,omitempty"`
	Seasons  []SyncHistoryParamsItem `json:"seasons,omitempty"`
	Episodes []SyncHistoryParamsItem `json:"episodes,omitempty"`
	Ids      []int64                 `json:"ids,omitempty"`
}

func (c APIClient) RemoveFromHistory(params *RemoveFromHistoryParams) (request.APIResponse[RemoveFromHistoryData], error) {
	params.JSON = params
	response := RemoveFromHistoryData{}
	res, err := c.Request("POST", "/sync/history/remove", params, &response)
	return request.NewAPIResponse(res, response), err
}
