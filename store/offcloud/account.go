package offcloud

import "time"

type GetAccountParams struct {
	Ctx
}

type GetAccountData struct {
	ResponseContainer
	APIKey             string `json:"apiKey"`
	Email              string `json:"email"`
	EmailNotifications []any  `json:"emailNotifications"`
	ReferralCode       string `json:"referralCode"`
}

func (c APIClient) GetAccount(params *GetAccountParams) (APIResponse[GetAccountData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetAccountData{}
	res, err := c.Request("POST", "/account/get", params, response)
	return newAPIResponse(res, *response), err
}

type GetAccountStatsParams struct {
	Ctx
}

type GetAccountStatsData struct {
	ResponseContainer
	AdditionalTags map[string]any `json:"additionalTags"`
	ExpirationDate int64          `json:"expirationDate"`
	MembershipType any            `json:"membershipType"`
	Revenue        int            `json:"revenue"`
}

func (c APIClient) GetAccountStats(params *GetAccountStatsParams) (APIResponse[GetAccountStatsData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetAccountStatsData{}
	res, err := c.Request("POST", "/account/stats", params, response)
	return newAPIResponse(res, *response), err
}

type GetUserEmailParams struct {
	Ctx
}

type GetUserEmailData struct {
	ResponseContainer
	CreatedOn  time.Time `json:"createdOn"`
	Email      string    `json:"email"`
	ProfilePic string    `json:"profilePic"`
	UserId     string    `json:"userId"`
}

func (c APIClient) GetUserEmail(params *GetUserEmailParams) (APIResponse[GetUserEmailData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetUserEmailData{}
	res, err := c.Request("POST", "/auth/user/email", params, response)
	return newAPIResponse(res, *response), err
}
