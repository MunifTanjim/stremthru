package serializd

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type ListTopShowsData struct {
	ResponseError
	Results []struct {
		AverageRating float32 `json:"averageRating"`
		ShowDetails   Show    `json:"showDetails"`
	} `json:"results"`
	TotalPages int `json:"totalPages"`
}

type ListTopShowsParams struct {
	Ctx
	Page int
}

func (c APIClient) ListTopShows(params *ListTopShowsParams) (request.APIResponse[ListTopShowsData], error) {
	params.Query = &url.Values{}
	if params.Page > 0 {
		params.Query.Set("page", strconv.Itoa(params.Page))
	}
	response := ListTopShowsData{}
	res, err := c.Request("GET", "/api/search/top_shows", params, &response)
	return request.NewAPIResponse(res, response), err
}
