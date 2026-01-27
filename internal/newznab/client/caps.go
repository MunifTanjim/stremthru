package newznab_client

import (
	"net/url"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type GetCapsParams struct {
	Ctx
}

func (c *Client) getCaps(params *GetCapsParams) (request.APIResponse[Caps], error) {
	params.Query = &url.Values{
		"t": {string(FunctionCaps)},
	}

	var resp Response[Caps]
	res, err := c.Request("GET", "/api", params, &resp)
	return request.NewAPIResponse(res, resp.Data), err
}

func (c *Client) GetCaps() (Caps, error) {
	return c.caps.Get()
}
