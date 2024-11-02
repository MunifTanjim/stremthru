package premiumize

import (
	"encoding/json"
	"net/url"
)

type CheckCacheData struct {
	Response   []bool        `json:"response"`
	Transcoded []bool        `json:"transcoded"`
	Filename   []string      `json:"filename"`
	Filesize   []json.Number `json:"filesize"`
}

type checkCacheData struct {
	ResponseContainer
	CheckCacheData
}

type CheckCacheParams struct {
	Ctx
	Items []string
}

func (c APIClient) CheckCache(params *CheckCacheParams) (APIResponse[CheckCacheData], error) {
	params.Form = &url.Values{
		"items[]": params.Items,
	}

	response := &checkCacheData{}
	res, err := c.Request("GET", "/cache/check", params, response)
	return newAPIResponse(res, response.CheckCacheData), err
}
