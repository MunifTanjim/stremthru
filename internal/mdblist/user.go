package mdblist

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
