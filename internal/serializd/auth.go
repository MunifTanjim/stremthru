package serializd

import "github.com/MunifTanjim/stremthru/internal/request"

type LoginData struct {
	ResponseError
	Token    string `json:"token"`
	Username string `json:"username"`
}

type LoginParams struct {
	Ctx
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c APIClient) Login(params *LoginParams) (request.APIResponse[LoginData], error) {
	params.JSON = map[string]string{
		"email":    params.Email,
		"password": params.Password,
	}
	response := LoginData{}
	res, err := c.Request("POST", "/api/login", params, &response)
	return request.NewAPIResponse(res, response), err
}
