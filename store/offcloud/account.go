package offcloud

type GetAccountInfoParams struct {
	Ctx
}

type GetAccountInfoData struct {
	ResponseContainer
	UserID         string `json:"user_id"`
	IsPremium      bool   `json:"is_premium"`
	ExpirationDate string `json:"expiration_date,omitempty"` // 2026-12-31
	CanDownload    bool   `json:"can_download"`
}

func (c APIClient) GetAccountInfo(params *GetAccountInfoParams) (APIResponse[GetAccountInfoData], error) {
	response := GetAccountInfoData{}
	res, err := c.Request("GET", "/api/account/info", params, &response)
	return newAPIResponse(res, response), err
}
