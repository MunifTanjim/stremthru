package trakt

import (
	"net/url"
	"time"
)

type RetrieveSettingsData struct {
	ResponseError
	User struct {
		Username string `json:"username"`
		Private  bool   `json:"private"`
		Deleted  bool   `json:"deleted"`
		Name     string `json:"name"`
		Vip      bool   `json:"vip"`
		VipEp    bool   `json:"vip_ep"`
		Director bool   `json:"director"`
		Ids      struct {
			Slug string `json:"slug"`
			UUID string `json:"uuid"`
		} `json:"ids"`
		JointedAt time.Time `json:"joined_at"`
		Location  string    `json:"location"`
		About     string    `json:"about"`
		Gender    string    `json:"gender"` // male
		Age       int       `json:"age"`
		Images    struct {
			Avatar struct {
				Full string `json:"full"`
			} `json:"avatar"`
		} `json:"images"`
		VipOg         bool     `json:"vip_og"`
		VipYears      int      `json:"vip_years"`
		VipCoverImage struct{} `json:"vip_cover_image"`
	} `json:"user"`
	Account struct {
		Timezone   string   `json:"timezone"`
		DateFormat string   `json:"date_format"`
		Time24Hr   bool     `json:"time_24hr"`
		CoverImage struct{} `json:"cover_image"`
		Token      struct{} `json:"token"`
		DisplayAds bool     `json:"display_ads"`
	}
	Connections struct {
		Facebook  bool `json:"facebook"`
		Twitter   bool `json:"twitter"`
		Mastodon  bool `json:"mastodon"`
		Google    bool `json:"google"`
		Tumblr    bool `json:"tumblr"`
		Medium    bool `json:"medium"`
		Slack     bool `json:"slack"`
		Apple     bool `json:"apple"`
		Dropbox   bool `json:"dropbox"`
		Microsoft bool `json:"microsoft"`
	} `json:"connections"`
	SharingText struct {
		Watching string   `json:"watching"`
		Watched  string   `json:"watched"`
		Rated    struct{} `json:"rated"`
	} `json:"sharing_text"`
	Limits struct {
		List struct {
			Count     int `json:"count"`
			ItemCount int `json:"item_count"`
		} `json:"list"`
		Watchlist struct {
			ItemCount int `json:"item_count"`
		} `json:"watchlist"`
		Favorites struct {
			ItemCount int `json:"item_count"`
		} `json:"favorites"`
		Search struct {
			RecentCount int `json:"recent_count"`
		} `json:"search"`
		Collection struct {
			ItemCount int `json:"item_count"`
		} `json:"collection"`
		Notes struct {
			ItemCount int `json:"item_count"`
		} `json:"notes"`
		Recommendations struct {
			ItemCount int `json:"item_count"`
		} `json:"recommendations"`
	} `json:"limits"`
	Permissions struct {
		Commenting bool `json:"commenting"`
		Liking     bool `json:"liking"`
		Following  bool `json:"following"`
	}
}

type RetrieveSettingsParams struct {
	Ctx
}

func (c APIClient) RetrieveSettings(params *RetrieveSettingsParams) (APIResponse[RetrieveSettingsData], error) {
	response := RetrieveSettingsData{}
	res, err := c.Request("GET", "/users/settings", params, &response)
	return newAPIResponse(res, response), err
}

type FetchPersonalListData struct {
	ResponseError
	List
}

type FetchPersonalListParams struct {
	Ctx
	UserId string
	ListId string
}

func (c APIClient) FetchPersonalList(params *FetchPersonalListParams) (APIResponse[List], error) {
	response := FetchPersonalListData{}
	res, err := c.Request("GET", "/users/"+params.UserId+"/lists/"+params.ListId, params, &response)
	return newAPIResponse(res, response.List), err
}

type FetchHiddenItemsDataItem struct {
	HiddenAt string   `json:"hidden_at"`
	Type     ItemType `json:"type"`
	Show     *struct {
		AiredEpisodes int         `json:"aired_episodes"`
		Title         string      `json:"title"`
		Year          int         `json:"year"`
		Ids           ListItemIds `json:"ids"`
	} `json:"show,omitempty"`
}

type FetchHiddenItemsData = listResponseData[FetchHiddenItemsDataItem]

type FetchHiddenItemsParams struct {
	Ctx
	Section string // calendar / progress_watched / progress_watched_reset / progress_collected / recommendations / comments / dropped
	Type    ItemType
}

func (c APIClient) FetchHiddenItems(params *FetchHiddenItemsParams) (APIResponse[[]FetchHiddenItemsDataItem], error) {
	query := url.Values{
		"page":  []string{"1"},
		"limit": []string{"1000"},
		"type":  []string{params.Type},
	}
	params.Query = &query
	response := FetchHiddenItemsData{}
	res, err := c.Request("GET", "/users/hidden/"+params.Section, params, &response)
	return newAPIResponse(res, response.data), err
}
