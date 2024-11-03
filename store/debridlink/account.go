package debridlink

type GetAccountInfoDataSettings struct {
	Https         bool   `json:"https"`
	ThemeDark     bool   `json:"themeDark"`
	HideOldLinks  bool   `json:"hideOldLinks"`
	Cdn           string `json:"cdn"`
	TwofaType     string `json:"twofaType"`
	EmailsNews    bool   `json:"emailsNews"`
	EmailsAccount bool   `json:"emailsAccount"`
	EmailsSupport bool   `json:"emailsSupport"`
}

type GetAccountInfoData struct {
	Email             string                     `json:"email"`
	EmailVerified     bool                       `json:"emailVerified"`
	AccountType       int                        `json:"accountType"`
	PremiumLeft       int                        `json:"premiumLeft"`
	Pts               int                        `json:"pts"`
	Trafficshare      int                        `json:"trafficshare"`
	VouchersUrl       string                     `json:"vouchersUrl"`
	EditPasswordUrl   string                     `json:"editPasswordUrl"`
	EditEmailUrl      string                     `json:"editEmailUrl"`
	ViewSessidUrl     string                     `json:"viewSessidUrl"`
	UpgradeAccountUrl string                     `json:"upgradeAccountUrl"`
	RegisterDate      string                     `json:"registerDate"`
	ServerDetected    bool                       `json:"serverDetected"`
	Settings          GetAccountInfoDataSettings `json:"settings"`
	AvatarUrl         string                     `json:"avatarUrl"`
	Username          string                     `json:"username"`
}

type GetAccountInfoParams struct {
	Ctx
}

func (c APIClient) GetAccountInfo(params *GetAccountInfoParams) (APIResponse[GetAccountInfoData], error) {
	response := &Response[GetAccountInfoData]{}
	res, err := c.Request("GET", "/v2/account/infos", params, response)
	return newAPIResponse(res, response.Value), err
}
