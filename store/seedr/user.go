package seedr

import "github.com/MunifTanjim/stremthru/internal/request"

type GetUserData struct {
	ResponseContainer
	Profile struct {
		Id        int    `json:"id"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		CreatedAt int64  `json:"created_at"`
		LastLogin int64  `json:"last_login"`
	} `json:"profile"`
	Account struct {
		IsPremium bool `json:"is_premium"`
		Storage   struct {
			Used  int64 `json:"used"`
			Limit int64 `json:"limit"`
		} `json:"storage"`
		Features struct {
			MaxTorrents         int `json:"max_torrents"`
			ActiveTorents       int `json:"active_torrents"`
			MaxWishlists        int `json:"max_wishlists"`
			ConcurrentDownloads int `json:"concurrent_downloads"`
		}
	}
}

type GetUserParams struct {
	Ctx
}

func (c APIClient) GetUser(params *GetUserParams) (request.APIResponse[GetUserData], error) {
	response := &GetUserData{}
	res, err := c.Request("GET", "/v0.1/p/user", params, response)
	return request.NewAPIResponse(res, *response), err
}
