package easydebrid

type GetUserDetailsData struct {
	ResponseContainer
	Id        string `json:"id"`
	PaidUntil int64  `json:"paid_until"` // unix seconds
}

type GetUserDetailsParams struct {
	Ctx
}

func (c APIClient) GetUserDetails(params *GetUserDetailsParams) (APIResponse[GetUserDetailsData], error) {
	response := &GetUserDetailsData{}
	res, err := c.Request("GET", "/v1/user/details", params, response)
	return newAPIResponse(res, *response), err
}
