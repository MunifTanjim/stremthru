package mdblist

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/core"
)

type Rating struct {
	Score  int     `json:"score"`
	Value  float32 `json:"value,omitempty"`
	Votes  int     `json:"votes,omitempty"`
	Source string  `json:"source"` // imdb / mdblist / tmdb
}

type Item struct {
	Adult       int     `json:"adult"`   // 0 / 1
	Country     string  `json:"country"` // us
	Description string  `json:"description,omitempty"`
	Genres      []Genre `json:"genres,omitempty"`
	Id          int     `json:"id"`
	Ids         struct {
		MDBList string `json:"mdblist,omitempty"`
		IMDB    string `json:"imdb,omitempty"`
		TMDB    int    `json:"tmdb,omitempty"`
		TVDB    int    `json:"tvdb,omitempty"`
	} `json:"ids"`
	ImdbId             string    `json:"imdb_id"`
	Language           string    `json:"language"`  // en
	Mediatype          MediaType `json:"mediatype"` // movie / show
	Poster             string    `json:"poster,omitempty"`
	Rank               int       `json:"rank"`
	Ratings            []Rating  `json:"ratings,omitempty"`
	ReleaseData        string    `json:"release_date,omitempty"` // YYYY-MM-DD
	ReleaseYear        int       `json:"release_year"`
	Runtime            int       `json:"runtime,omitempty"`
	SpokenLanguage     string    `json:"spoken_language"`
	Status             string    `json:"status"` // released
	Title              string    `json:"title"`
	TotalAiredEpisodes int       `json:"total_aired_episodes,omitempty"`
	TvdbId             int       `json:"tvdb_id,omitempty"`
}

type FetchListItemsData = []Item

type listResponseData[T any] struct {
	ResponseContainer
	data []T
}

func (d *listResponseData[T]) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		d.ResponseContainer = rerr
		return nil
	}

	var items []T
	if err := json.Unmarshal(data, &items); err == nil {
		d.data = items
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

type PageSort = string  // rank / score / usort / score_average / released / releasedigital / imdbrating / imdbvotes / last_air_date / imdbpopular / tmdbpopular / rogerebert / rtomatoes / rtaudience / metacritic / myanimelist / letterrating / lettervotes / budget / revenue / runtime / title / added / random
type PageOrder = string // asc / desc

type FetchListItemsParams struct {
	Ctx
	ListId      int
	Limit       int
	Offset      int
	FilterGenre Genre
	Sort        PageSort
	Order       PageOrder // asc / desc
}

func (c APIClient) FetchListItems(params *FetchListItemsParams) (APIResponse[FetchListItemsData], error) {
	query := url.Values{}
	if params.Limit != 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset != 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	query.Set("append_to_response", "genres,poster,description") // ratings
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Order != "" {
		query.Set("order", params.Order)
	}
	query.Set("unified", "true")
	params.Query = &query

	response := &listResponseData[Item]{}
	res, err := c.Request("GET", "/lists/"+strconv.Itoa(params.ListId)+"/items", params, response)
	return newAPIResponse(res, response.data), err
}

type FetchExternalListItemsParams struct {
	Ctx
	ListId      int
	Limit       int
	Offset      int
	FilterGenre Genre
	Sort        PageSort
	Order       PageOrder // asc / desc
}

func (c APIClient) FetchExternalListItems(params *FetchExternalListItemsParams) (APIResponse[FetchListItemsData], error) {
	query := url.Values{}
	if params.Limit != 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset != 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	query.Set("append_to_response", "genres,poster,description") // ratings
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Order != "" {
		query.Set("order", params.Order)
	}
	query.Set("unified", "true")
	params.Query = &query

	response := &listResponseData[Item]{}
	res, err := c.Request("GET", "/external/lists/"+strconv.Itoa(params.ListId)+"/items", params, response)
	return newAPIResponse(res, response.data), err
}
