package trakt

import (
	"net/url"
	"time"
)

type retrieveSettingsDataBrowsing struct {
	WatchPopupAction         string `json:"watch_popup_action"`
	HideWatchingNow          bool   `json:"hide_watching_now"`
	ListPopupAction          string `json:"list_popup_action"`
	WeekStartDay             string `json:"week_start_day"`
	WatchAfterRating         string `json:"watch_after_rating"`
	WatchOnlyOnce            bool   `json:"watch_only_once"`
	OtherSiteRatings         bool   `json:"other_site_ratings"`
	ReleaseDateIgnoreRuntime bool   `json:"release_date_ignore_runtime"`
	DisplayEarlyRatings      bool   `json:"display_early_ratings"`
	HideEpisodeTypeTags      bool   `json:"hide_episode_type_tags"`
	HideUnsavedFiltersPrompt bool   `json:"hide_unsaved_filters_prompt"`
	Spoilers                 struct {
		Episodes string `json:"episodes"`
		Shows    string `json:"shows"`
		Movies   string `json:"movies"`
		Comments string `json:"comments"`
		Ratings  string `json:"ratings"`
		Actors   string `json:"actors"`
	} `json:"spoilers"`
	Calendar struct {
		Period       string `json:"period"`
		StartDay     string `json:"start_day"`
		Layout       string `json:"layout"`
		ImageType    any    `json:"image_type"`
		HideSpecials bool   `json:"hide_specials"`
		Autoscroll   bool   `json:"autoscroll"`
	} `json:"calendar"`
	Progress struct {
		OnDeck struct {
			Sort           string `json:"sort"`
			SortHow        string `json:"sort_how"`
			Refresh        bool   `json:"refresh"`
			SimpleProgress bool   `json:"simple_progress"`
			OnlyFavorites  bool   `json:"only_favorites"`
		} `json:"on_deck"`
		Watched struct {
			Refresh            bool   `json:"refresh"`
			SimpleProgress     bool   `json:"simple_progress"`
			IncludeSpecials    bool   `json:"include_specials"`
			IncludeWatchlisted bool   `json:"include_watchlisted"`
			IncludeCollected   bool   `json:"include_collected"`
			Sort               string `json:"sort"`
			SortHow            string `json:"sort_how"`
			UseLastActivity    bool   `json:"use_last_activity"`
			GridView           bool   `json:"grid_view"`
		} `json:"watched"`
		Collected struct {
			Refresh            bool   `json:"refresh"`
			SimpleProgress     bool   `json:"simple_progress"`
			IncludeSpecials    bool   `json:"include_specials"`
			IncludeWatchlisted bool   `json:"include_watchlisted"`
			IncludeWatched     bool   `json:"include_watched"`
			Sort               string `json:"sort"`
			SortHow            string `json:"sort_how"`
			UseLastActivity    bool   `json:"use_last_activity"`
			GridView           bool   `json:"grid_view"`
		} `json:"collected"`
	} `json:"progress"`
	Watchnow struct {
		Country       string `json:"country"`
		Favorites     []any  `json:"favorites"`
		OnlyFavorites bool   `json:"only_favorites"`
	} `json:"watchnow"`
	DarkKnight string `json:"dark_knight"`
	AppTheme   string `json:"app_theme"`
	Welcome    struct {
		CompletedAt time.Time `json:"completed_at"`
		ExitStep    string    `json:"exit_step"`
	} `json:"welcome"`
	Genres struct {
		Favorites any `json:"favorites"`
		Disliked  any `json:"disliked"`
	} `json:"genres"`
	Comments struct {
		BlockedUids []any `json:"blocked_uids"`
	} `json:"comments"`
	Recommendations struct {
		IgnoreCollected   bool `json:"ignore_collected"`
		IgnoreWatchlisted bool `json:"ignore_watchlisted"`
	} `json:"recommendations"`
	Rewatching struct {
		AdjustPercentage bool `json:"adjust_percentage"`
	} `json:"rewatching"`
	Profile struct {
		Favorites struct {
			SortBy  string `json:"sort_by"`
			SortHow string `json:"sort_how"`
		} `json:"favorites"`
		MostWatchedShows struct {
			SortBy string `json:"sort_by"`
			Tab    string `json:"tab"`
		} `json:"most_watched_shows"`
		MostWatchedMovies struct {
			SortBy string `json:"sort_by"`
			Tab    string `json:"tab"`
		} `json:"most_watched_movies"`
	} `json:"profile"`
	Search struct {
		ImageType     any   `json:"image_type"`
		RecentQueries []any `json:"recent_queries"`
	} `json:"search"`
	Yir struct {
		ShowsMostPlayed  string `json:"shows_most_played"`
		MoviesMostPlayed string `json:"movies_most_played"`
	} `json:"yir"`
}

type retrieveSettingsDataSharing struct {
	Email struct {
		NewFollower    bool `json:"new_follower"`
		CommentMention bool `json:"comment_mention"`
		CommentReply   bool `json:"comment_reply"`
		CommentLike    bool `json:"comment_like"`
		ListComment    bool `json:"list_comment"`
		ListLike       bool `json:"list_like"`
	} `json:"email"`
	App struct {
		NewFollower          bool `json:"new_follower"`
		CommentMention       bool `json:"comment_mention"`
		CommentReply         bool `json:"comment_reply"`
		CommentLike          bool `json:"comment_like"`
		ListComment          bool `json:"list_comment"`
		ListLike             bool `json:"list_like"`
		PendingCollaboration bool `json:"pending_collaboration"`
		WeeklyDigest         bool `json:"weekly_digest"`
		Mir                  bool `json:"mir"`
	} `json:"app"`
}

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
	Browsing    *retrieveSettingsDataBrowsing `json:"browsing,omitempty"`
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
	Sharing *retrieveSettingsDataSharing `json:"sharing,omitempty"`
	Limits  struct {
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
	Extended string // browsing / sharing
}

func (c APIClient) RetrieveSettings(params *RetrieveSettingsParams) (APIResponse[RetrieveSettingsData], error) {
	if params.Extended != "" {
		params.Query = &url.Values{}
		params.Query.Set("extended", params.Extended)
	}
	response := RetrieveSettingsData{}
	res, err := c.Request("GET", "/users/settings", params, &response)
	return newAPIResponse(res, response), err
}

type FetchUserListData struct {
	ResponseError
	List
}

type FetchUserListParams struct {
	Ctx
	UserId string
	ListId string
}

func (c APIClient) FetchUserList(params *FetchUserListParams) (APIResponse[List], error) {
	response := FetchUserListData{}
	path := "/lists/" + params.ListId
	if params.UserId != "" {
		path = "/users/" + params.UserId + path
	}
	res, err := c.Request("GET", path, params, &response)
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
