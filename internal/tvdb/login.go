package tvdb

import "github.com/MunifTanjim/stremthru/internal/request"

type LoginData struct {
	Token string `json:"token"`
}

type LoginParams struct {
	Ctx
	APIKey string `json:"apikey"`
	PIN    string `json:"pin,omitempty"`
}

func (c APIClient) Login(params *LoginParams) (request.APIResponse[LoginData], error) {
	params.JSON = params
	response := Response[LoginData]{}
	res, err := c.Request("POST", "/login", params, &response)
	return request.NewAPIResponse(res, response.Data), err
}
