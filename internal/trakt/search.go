package trakt

import (
	"net/url"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type IdType string

const (
	IdTypeTrakt IdType = "trakt"
	IdTypeIMDB  IdType = "imdb"
	IdTypeTMDB  IdType = "tmdb"
	IdTypeTVDB  IdType = "tvdb"
)

type LookupIdDataItem struct {
	Type  ItemType     `json:"type"` // movie/show/episode/person/list
	Score int          `json:"score"`
	Movie *minimalItem `json:"movie,omitempty"`
	Show  *minimalItem `json:"show,omitempty"`
}

type LookupIdData = []LookupIdDataItem

type LookupIdParams struct {
	Ctx
	IdType IdType
	Id     string
	Type   ItemType // movie/show/episode/person/list
}

func (c APIClient) LookupId(params *LookupIdParams) (request.APIResponse[LookupIdData], error) {
	path := "/search/" + string(params.IdType) + "/" + params.Id
	if params.Type != "" {
		params.Query = &url.Values{
			"type": []string{string(params.Type)},
		}
	}
	response := paginatedResponseData[LookupIdDataItem]{}
	res, err := c.Request("GET", path, params, &response)
	return request.NewAPIResponse(res, response.data), err
}
