package premiumize

type GetAccountInfoData struct {
	CustomerId   string  `json:"customer_id"`
	PremiumUntil int     `json:"premium_until"`
	LimitUsed    float32 `json:"limit_used"`
	SpaceUsed    int     `json:"space_used"`
}

type getAccountInfoData struct {
	ResponseContainer
	GetAccountInfoData
}

type GetAccountInfoParams struct {
	Ctx
}

func (c APIClient) GetAccountInfo(params *GetAccountInfoParams) (APIResponse[GetAccountInfoData], error) {
	response := &getAccountInfoData{}
	res, err := c.Request("GET", "/account/info", params, response)
	return newAPIResponse(res, response.GetAccountInfoData), err
}
