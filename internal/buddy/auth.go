package buddy

import (
	"net/http"
)

type CheckAuthData struct {
}

type CheckAuthParams struct {
	Ctx
	Token string
}

func (c APIClient) CheckAuth(params *CheckAuthParams) (APIResponse[CheckAuthData], error) {
	params.Headers = &http.Header{
		"X-StremThru-Buddy-Token": []string{params.Token},
	}
	response := &Response[CheckAuthData]{}
	res, err := c.Request("GET", "/v0/auth/check", params, response)
	return newAPIResponse(res, response.Data), err
}
