package letterboxd

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type FetchMemberWatchlistData struct {
	ResponseError
	Next  string        `json:"next"`
	Items []FilmSummary `json:"items"`
}

type FetchMemberWatchlistParams struct {
	Ctx
	Id      string
	Cursor  string
	PerPage int // default 20, max 100
}

func (c *APIClient) FetchMemberWatchlist(params *FetchMemberWatchlistParams) (request.APIResponse[FetchMemberWatchlistData], error) {
	query := url.Values{}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}
	if params.PerPage > 0 {
		if params.PerPage > 100 {
			panic("perPage maximum is 100")
		}
		query.Set("perPage", strconv.Itoa(params.PerPage))
	}
	params.Query = &query
	response := FetchMemberWatchlistData{}
	res, err := c.Request("GET", "/v0/member/"+params.Id+"/watchlist", params, &response)
	return request.NewAPIResponse(res, response), err
}

type MemberIdentifier struct {
	Id string `json:"id"`
}

type MemberStatisticsCounts struct {
	FilmLikes            int `json:"filmLikes"`
	ListLikes            int `json:"listLikes"`
	ReviewLikes          int `json:"reviewLikes"`
	StoryLikes           int `json:"storyLikes"`
	Watches              int `json:"watches"`
	Ratings              int `json:"ratings"`
	Reviews              int `json:"reviews"`
	DiaryEntries         int `json:"diaryEntries"`
	DiaryEntriesThisYear int `json:"diaryEntriesThisYear"`
	FilmsInDiaryThisYear int `json:"filmsInDiaryThisYear"`
	FilmsInDiaryLastYear int `json:"filmsInDiaryLastYear"`
	Watchlist            int `json:"watchlist"`
	Lists                int `json:"lists"`
	UnpublishedLists     int `json:"unpublishedLists,omitempty"`
	AccessedSharedLists  int `json:"accessedSharedLists,omitempty"`
	Followers            int `json:"followers"`
	Following            int `json:"following"`
	ListTags             int `json:"listTags"`
	FilmTags             int `json:"filmTags"`
}

type RatingsHistogramBar struct {
	Rating           float64 `json:"rating"`           // 0.5 - 5.0
	NormalizedWeight float64 `json:"normalizedWeight"` // 0.0 - 1.0
	Count            int     `json:"count"`
}

type FetchMemberStatisticsData struct {
	ResponseError
	Member           MemberIdentifier       `json:"member"`
	Counts           MemberStatisticsCounts `json:"counts"`
	RatingsHistogram []RatingsHistogramBar  `json:"ratingsHistogram"`
	YearsInReview    []int                  `json:"yearsInReview"` // Only supported for paying members
}

type FetchMemberStatisticsParams struct {
	Ctx
	Id string
}

func (c *APIClient) FetchMemberStatistics(params *FetchMemberStatisticsParams) (request.APIResponse[FetchMemberStatisticsData], error) {
	response := FetchMemberStatisticsData{}
	res, err := c.Request("GET", "/v0/member/"+params.Id+"/statistics", params, &response)
	return request.NewAPIResponse(res, response), err
}
