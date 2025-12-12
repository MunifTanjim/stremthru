package mdblist

import (
	"net/url"
	"strconv"
	"time"
)

type WatchlistItem struct {
	Item
	WatchlistedAt time.Time `json:"watchlisted_at"`
}

type FetchWatchlistItemsParams struct {
	Ctx
	Limit       int
	Offset      int
	FilterGenre Genre
	Sort        PageSort  // score / score_average / released / releasedigital / imdbrating / imdbvotes / last_air_date / imdbpopular / tmdbpopular / rogerebert / rtomatoes / rtaudience / metacritic / myanimelist / letterrating / lettervotes / budget / revenue / runtime / title / added / random
	Order       PageOrder // asc / desc
}

func (c APIClient) FetchWatchlistItems(params *FetchWatchlistItemsParams) (APIResponse[[]WatchlistItem], error) {
	query := url.Values{}
	if params.Limit != 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset != 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	query.Set("append_to_response", "genre,poster")
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Order != "" {
		query.Set("order", params.Order)
	}
	query.Set("unified", "true")
	params.Query = &query

	response := &listResponseData[WatchlistItem]{}
	res, err := c.Request("GET", "/watchlist/items", params, response)
	return newAPIResponse(res, response.data), err
}
