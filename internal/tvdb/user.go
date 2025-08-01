package tvdb

import "github.com/MunifTanjim/stremthru/internal/request"

type GetUserData struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Language string `json:"language"`
}

type GetUserParams struct {
	Ctx
}

func (c APIClient) GetUser(params *GetUserParams) (request.APIResponse[GetUserData], error) {
	response := Response[GetUserData]{}
	res, err := c.Request("GET", "/user", params, &response)
	return request.NewAPIResponse(res, response.Data), err
}
