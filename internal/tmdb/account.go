package tmdb

type GetAccountDetailsData struct {
	ResponseError
	Avatar struct {
		Gravatar struct {
			Hash string `json:"hash"`
		} `json:"gravatar"`
		TMDB struct {
			AvatarPath string `json:"avatar_path"`
		} `json:"tmdb"`
	} `json:"avatar"`
	Id           int64  `json:"id"`
	ISO6381      string `json:"iso_639_1"`
	ISO31661     string `json:"iso_3166_1"`
	Name         string `json:"name"`
	IncludeAdult bool   `json:"include_adult"`
	Username     string `json:"username"`
}

type GetAccountDetailsParams struct {
	Ctx
}

func (c APIClient) GetAccountDetails(params *GetAccountDetailsParams) (APIResponse[GetAccountDetailsData], error) {
	response := GetAccountDetailsData{}
	res, err := c.Request("GET", "/3/account", params, &response)
	return newAPIResponse(res, response), err
}
