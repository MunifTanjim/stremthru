package tvdb

import (
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type Alias struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type List struct {
	Id                   int        `json:"id"`
	Name                 string     `json:"name"`
	Overview             string     `json:"overview"`
	URL                  string     `json:"url"` // slug
	IsOfficial           bool       `json:"isOfficial"`
	NameTranslations     []string   `json:"nameTranslations"`
	OverviewTranslations []string   `json:"overviewTranslations"`
	Aliases              []Alias    `json:"aliases"`
	Score                int64      `json:"score"`
	Image                string     `json:"image"`
	ImageIsFallback      bool       `json:"imageIsFallback"`
	RemoteIds            []RemoteId `json:"remoteIds"`
	Tags                 any        `json:"tags"`
}

type ListEntity struct {
	Order    int `json:"order"`
	SeriesId int `json:"seriesId"`
	MovieId  int `json:"movieId"`
}

type ExtendedList struct {
	List
	Entities []ListEntity `json:"entities"`
}

type FetchListParams struct {
	Ctx
	Id   int
	Slug string
}

func (c APIClient) FetchList(params *FetchListParams) (request.APIResponse[List], error) {
	var endpoint string
	if params.Id != 0 {
		endpoint = "/lists/" + strconv.Itoa(params.Id)
	} else if params.Slug != "" {
		endpoint = "/lists/slug/" + params.Slug
	}
	response := Response[List]{}
	res, err := c.Request("GET", endpoint, params, &response)
	return request.NewAPIResponse(res, response.Data), err
}

type FetchExtendedListParams struct {
	Ctx
	Id int
}

func (c APIClient) FetchExtendedList(params *FetchExtendedListParams) (request.APIResponse[ExtendedList], error) {
	response := Response[ExtendedList]{}
	res, err := c.Request("GET", "/lists/"+strconv.Itoa(params.Id)+"/extended", params, &response)
	return request.NewAPIResponse(res, response.Data), err
}
