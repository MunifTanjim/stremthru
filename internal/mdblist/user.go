package mdblist

import (
	"encoding/json"

	"github.com/MunifTanjim/stremthru/core"
)

type GetMyLimitsData struct {
	ResponseContainer
	APIRequests      int    `json:"api_requests"`
	APIRequestsCount int    `json:"api_requests_count"`
	UserId           int    `json:"user_id"`
	Username         string `json:"username"`
	PatronStatus     string `json:"patron_status,omitempty"` // active_patron
	PatreonPledge    int    `json:"patreon_pledge,omitempty"`
}

type GetMyLimitsParams struct {
	Ctx
}

func (c APIClient) GetMyLimits(params *GetMyLimitsParams) (APIResponse[GetMyLimitsData], error) {
	response := &GetMyLimitsData{}
	res, err := c.Request("GET", "/user", params, response)
	return newAPIResponse(res, *response), err
}

type GetMyListsData []List

type getMyListsData struct {
	ResponseContainer
	data GetMyListsData
}

func (d *getMyListsData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		d.ResponseContainer = rerr
		return nil
	}

	var items GetMyListsData
	if err := json.Unmarshal(data, &items); err == nil {
		d.data = items
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

type GetMyListsParams struct {
	Ctx
}

func (c APIClient) GetMyLists(params *GetMyListsParams) (APIResponse[GetMyListsData], error) {
	response := &getMyListsData{}
	res, err := c.Request("GET", "/lists/user", params, response)
	return newAPIResponse(res, response.data), err
}
