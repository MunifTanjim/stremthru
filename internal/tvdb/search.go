package tvdb

import "github.com/MunifTanjim/stremthru/internal/request"

type SearchByRemoteIdData struct {
	Movie  *Movie
	Series *Series
}

type SearchByRemoteIdParams struct {
	Ctx
	RemoteId string
}

func (c APIClient) SearchByRemoteId(params *SearchByRemoteIdParams) (request.APIResponse[SearchByRemoteIdData], error) {
	response := Response[[]SearchByRemoteIdData]{}
	res, err := c.Request("GET", "/search/remoteid/"+params.RemoteId, params, &response)
	data := SearchByRemoteIdData{}
	if len(response.Data) > 0 {
		data = response.Data[0]
	}
	return request.NewAPIResponse(res, data), err
}
