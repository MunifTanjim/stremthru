package serializd

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type ListShowData struct {
	ResponseError
	Results    []ShowSummary `json:"results"`
	TotalPages int           `json:"totalPages"`
}

type ListPopularShowsParams struct {
	Ctx
	Page int
}

func (c APIClient) ListPopularShows(params *ListPopularShowsParams) (request.APIResponse[ListShowData], error) {
	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	response := ListShowData{}
	res, err := c.Request("GET", "/mobile/page/popular_shows", params, &response)
	return request.NewAPIResponse(res, response), err
}

type ListTrendingShowsParams struct {
	Ctx
	Page int
}

func (c APIClient) ListTrendingShows(params *ListTrendingShowsParams) (request.APIResponse[ListShowData], error) {
	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	response := ListShowData{}
	res, err := c.Request("GET", "/mobile/page/trending_shows", params, &response)
	return request.NewAPIResponse(res, response), err
}

type ListFeaturedShowsData struct {
	ResponseError
	Results []struct {
		ShowDetails ShowSummary `json:"showDetails"`
	} `json:"results"`
	TotalPages int `json:"totalPages"`
}

type ListFeaturedShowsParams struct {
	Ctx
	Page int
}

func (c APIClient) ListFeaturedShows(params *ListFeaturedShowsParams) (request.APIResponse[ListFeaturedShowsData], error) {
	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	response := ListFeaturedShowsData{}
	res, err := c.Request("GET", "/mobile/page/featured", params, &response)
	return request.NewAPIResponse(res, response), err
}

type ListAnticipatedShowsData struct {
	ResponseError
	Results []struct {
		SeasonDetails SeasonSummary `json:"seasonDetails"`
		ShowDetails   ShowSummary   `json:"showDetails"`
	} `json:"results"`
	TotalPages int `json:"totalPages"`
}

type ListAnticipatedShowsParams struct {
	Ctx
	Page int
}

func (c APIClient) ListAnticipatedShows(params *ListAnticipatedShowsParams) (request.APIResponse[ListAnticipatedShowsData], error) {
	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	response := ListAnticipatedShowsData{}
	res, err := c.Request("GET", "/mobile/page/anticipated", params, &response)
	return request.NewAPIResponse(res, response), err
}
