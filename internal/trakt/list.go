package trakt

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type ListPrivacy = string

const (
	ListPrivacyPublic  ListPrivacy = "public"
	ListPrivacyFriends ListPrivacy = "friends"
	ListPrivacyLink    ListPrivacy = "link"
	ListPrivacyPrivate ListPrivacy = "private"
)

type List struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Privacy        string    `json:"privacy"` // public / friends / link / private
	ShareLink      string    `json:"share_link"`
	Type           string    `json:"type"` // personal
	DisplayNumbers bool      `json:"display_numbers"`
	AllowComments  bool      `json:"allow_comments"`
	SortBy         string    `json:"sort_by"`  // rank
	SortHow        string    `json:"sort_how"` // asc
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ItemCount      int       `json:"item_count"`
	CommentCount   int       `json:"comment_count"`
	Likes          int       `json:"likes"`
	Ids            struct {
		Trakt int    `json:"trakt"`
		Slug  string `json:"slug"`
	} `json:"ids"`
	User struct {
		Username string `json:"username"`
		Private  bool   `json:"private"`
		Name     string `json:"name"`
		Vip      bool   `json:"vip"`
		VipEp    bool   `json:"vip_ep"`
		Ids      struct {
			Slug string `json:"slug"`
		} `json:"ids"`
	} `json:"user"`
}

type ListItemIds struct {
	Trakt  int    `json:"trakt"`
	Slug   string `json:"slug"`
	IMDB   string `json:"imdb,omitempty"`
	TMDB   int    `json:"tmdb,omitempty"`
	TVDB   int    `json:"tvdb,omitempty"`
	TVRage any    `json:"tv_rage,omitempty"`
}

type listItemCommon struct {
	Title                 string      `json:"title"`
	Year                  int         `json:"year"`
	Ids                   ListItemIds `json:"ids"`
	Tagline               string      `json:"tagline,omitempty"`
	Overview              string      `json:"overview,omitempty"`
	Runtime               int         `json:"runtime,omitempty"` // in minutes
	Certification         string      `json:"certification,omitempty"`
	Country               string      `json:"country,omitempty"` // us
	Status                string      `json:"status,omitempty"`  // released
	Rating                float32     `json:"rating,omitempty"`  // 0.0 - 10.0
	Votes                 int         `json:"votes,omitempty"`
	CommentCount          int         `json:"comment_count,omitempty"`
	Trailer               string      `json:"trailer,omitempty"`
	Homepage              string      `json:"homepage,omitempty"`
	UpdatedAt             *time.Time  `json:"updated_at,omitempty"`
	Language              string      `json:"language,omitempty"`
	Languages             []string    `json:"languages,omitempty"`
	AvailableTranslations []string    `json:"available_translations,omitempty"`
	Genres                []string    `json:"genres,omitempty"`
	OriginalTitle         string      `json:"original_title,omitempty"`
	Images                *struct {
		Fanart   []string `json:"fanart"`
		Poster   []string `json:"poster"`
		Logo     []string `json:"logo"`
		Clearart []string `json:"clearart"`
		Banner   []string `json:"banner"`
		Thumb    []string `json:"thumb"`
	} `json:"images,omitempty"`
}

type ListItemMovie struct {
	listItemCommon
	Released      string `json:"released,omitempty"` // YYYY-MM-DD
	AfterCredits  bool   `json:"after_credits,omitempty"`
	DuringCredits bool   `json:"during_credits,omitempty"`
}

type ListItemShow struct {
	listItemCommon
	FirstAired *time.Time `json:"first_aired,omitempty"`
	Airs       *struct {
		Day      string `json:"day"`      // e.g., "Monday"
		Time     string `json:"time"`     // e.g., "20:00"
		Timezone string `json:"timezone"` // e.g., "America/New_York"
	} `json:"airs,omitempty"`
	Network       string `json:"network,omitempty"`
	AiredEpisodes int    `json:"aired_episodes,omitempty"`
}

type ItemType = string

const (
	ItemTypeMovie   ItemType = "movie"
	ItemTypeShow    ItemType = "show"
	ItemTypeSeason  ItemType = "season"
	ItemTypeEpisode ItemType = "episode"
)

type listItemNextEpisode struct {
	Season  int
	Episode int
}

type ListItem struct {
	Rank        int                  `json:"rank"`
	Id          int64                `json:"id"`
	ListedAt    time.Time            `json:"listed_at"`
	Note        string               `json:"note,omitempty"`
	Type        ItemType             `json:"type"`
	Movie       *ListItemMovie       `json:"movie,omitempty"`
	Show        *ListItemShow        `json:"show,omitempty"`
	NextEpisode *listItemNextEpisode `json:"-"`
}

type FetchListItemsData = []ListItem

type listResponseData[T any] struct {
	ResponseError
	data []T
}

func (d *listResponseData[T]) UnmarshalJSON(data []byte) error {
	var rerr ResponseError

	if err := json.Unmarshal(data, &rerr); err == nil {
		d.ResponseError = rerr
		return nil
	}

	var items []T
	err := json.Unmarshal(data, &items)
	if err == nil {
		d.data = items
		return nil
	}

	e := core.NewAPIError("failed to parse response")
	e.Cause = err
	return e
}

type FetchUserListItemsParams struct {
	Ctx
	UserId   string
	ListId   string
	Type     []ItemType
	SortBy   string // rank / added / title / released / runtime / popularity / random / percentage / my_rating / watched / collected
	SortHow  string // asc / desc
	Extended string // images / full / full,images
}

func (c APIClient) FetchUserListItems(params *FetchUserListItemsParams) (APIResponse[FetchListItemsData], error) {
	params.Query = &url.Values{}
	if params.Extended != "" {
		params.Query.Set("extended", params.Extended)
	}

	response := listResponseData[ListItem]{}
	path := "/lists/" + params.ListId + "/items"
	if len(params.Type) > 0 {
		path += "/" + strings.Join(params.Type, ",")
	} else if params.SortBy != "" {
		path += "/movie,show"
	}
	if params.SortBy != "" {
		path += "/" + params.SortBy
		if params.SortHow != "" {
			path += "/" + params.SortHow
		}
	}
	if params.UserId != "" {
		path = "/users/" + params.UserId + path
	}
	res, err := c.Request("GET", path, params, &response)
	return newAPIResponse(res, response.data), err
}

type dynamicListMeta struct {
	Endpoint      string
	BeforeRequest func(req *http.Request) error

	Id       string
	ItemType ItemType
	Name     string
	NoLimit  bool
	NoPage   bool

	HasPeriod bool
	Period    string

	HasUserId bool
	UserId    string
}

var dynamicListMetaById = map[string]dynamicListMeta{
	"shows/trending": {
		Endpoint: "/shows/trending",
		Name:     "Trending",
		ItemType: ItemTypeShow,
	},
	"shows/anticipated": {
		Endpoint: "/shows/anticipated",
		Name:     "Anticipated",
		ItemType: ItemTypeShow,
	},
	"shows/popular": {
		Endpoint: "/shows/popular",
		Name:     "Popular",
		ItemType: ItemTypeShow,
	},
	"shows/favorited": {
		Endpoint:  "/shows/favorited/{period}",
		HasPeriod: true,
		Name:      "Most Favorited",
		ItemType:  ItemTypeShow,
	},
	"shows/watched": {
		Endpoint:  "/shows/watched/{period}",
		HasPeriod: true,
		Name:      "Most Watched",
		ItemType:  ItemTypeShow,
	},
	"shows/collected": {
		Endpoint:  "/shows/collected/{period}",
		HasPeriod: true,
		Name:      "Most Collected",
		ItemType:  ItemTypeShow,
	},
	"shows/recommendations": {
		Endpoint: "/recommendations/shows",
		NoPage:   true,
		Name:     "Recommended",
		ItemType: ItemTypeShow,
		Id:       USER_SHOWS_RECOMMENDATIONS_ID,
	},

	"movies/trending": {
		Endpoint: "/movies/trending",
		Name:     "Trending",
		ItemType: ItemTypeMovie,
	},
	"movies/anticipated": {
		Endpoint: "/movies/anticipated",
		Name:     "Anticipated",
		ItemType: ItemTypeMovie,
	},
	"movies/popular": {
		Endpoint: "/movies/popular",
		Name:     "Popular",
		ItemType: ItemTypeMovie,
	},
	"movies/favorited": {
		Endpoint:  "/movies/favorited/{period}",
		HasPeriod: true,
		Name:      "Most Favorited",
		ItemType:  ItemTypeMovie,
	},
	"movies/watched": {
		Endpoint:  "/movies/watched/{period}",
		HasPeriod: true,
		Name:      "Most Watched",
		ItemType:  ItemTypeMovie,
	},
	"movies/collected": {
		Endpoint:  "/movies/collected/{period}",
		HasPeriod: true,
		Name:      "Most Collected",
		ItemType:  ItemTypeMovie,
	},
	"movies/boxoffice": {
		Endpoint: "/movies/boxoffice",
		NoPage:   true,
		Name:     "Weekend Box Office",
		ItemType: ItemTypeMovie,
	},
	"movies/recommendations": {
		Endpoint: "/recommendations/movies",
		NoPage:   true,
		Name:     "Recommended",
		ItemType: ItemTypeMovie,
		Id:       USER_MOVIES_RECOMMENDATIONS_ID,
	},

	"favorites": {
		Endpoint:  "/users/{user_id}/favorites",
		NoPage:    true,
		NoLimit:   true,
		Name:      "Favorites",
		HasUserId: true,
	},
	"watchlist": {
		Endpoint:  "/users/{user_id}/watchlist",
		NoPage:    true,
		NoLimit:   true,
		Name:      "Watchlist",
		HasUserId: true,
	},
	"progress": {
		Endpoint: "/sync/progress/up_next_nitro",
		BeforeRequest: func(req *http.Request) error {
			req.URL.Host = util.MustDecodeBase64("YXBpei50cmFrdC50dg==")
			req.Host = req.URL.Host
			req.Header.Set("Origin", util.MustDecodeBase64("aHR0cHM6Ly9hcHAudHJha3QudHY="))
			req.Header.Set("Referer", util.MustDecodeBase64("aHR0cHM6Ly9hcHAudHJha3QudHYv"))
			req.Header.Set("User-Agent", util.MustDecodeBase64("TW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTBfMTVfNykgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzEzOS4wLjAuMCBTYWZhcmkvNTM3LjM2"))
			return nil
		},
		Name:     "Up Next",
		ItemType: ItemTypeShow,
	},
}

type EpisodeType string

const (
	EpisodeTypeStandard EpisodeType = "standard"
)

type SyncProgressUpNextNitroItemEpisode struct {
	AvailableTranslations []string    `json:"available_translations"`
	CommentCount          int         `json:"comment_count"`
	EpisodeType           EpisodeType `json:"episode_type"`
	FirstAired            string      `json:"first_aired"`
	Ids                   ListItemIds `json:"ids"`
	Images                struct {
		Screenshot []string `json:"screenshot"`
	} `json:"images"`
	Number    int     `json:"number"`
	NumberAbs int     `json:"number_abs"`
	Overview  string  `json:"overview"`
	Rating    float32 `json:"rating"`
	Runtime   int     `json:"runtime"`
	Season    int     `json:"season"`
	Title     string  `json:"title"`
	UpdatedAt string  `json:"updated_at"`
	Votes     int     `json:"votes"`
}

func (ep SyncProgressUpNextNitroItemEpisode) IsAired() bool {
	if ep.FirstAired == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, ep.FirstAired)
	if err != nil {
		return false
	}
	return t.Before(time.Now())
}

type SyncProgressUpNextNitroItem struct {
	Progress struct {
		Aired         int                                `json:"aired"`
		Completed     int                                `json:"completed"`
		Hidden        int                                `json:"hidden"`
		LastEpisode   SyncProgressUpNextNitroItemEpisode `json:"last_episode"`
		LastWatchedAt string                             `json:"last_watched_at"`
		NextEpisode   SyncProgressUpNextNitroItemEpisode `json:"next_episode"`
		ResetAt       any                                `json:"reset_at"`
		Stats         struct {
			MinutesLeft    int `json:"minutes_left"`
			MinutesWatched int `json:"minutes_watched"`
			PlayCount      int `json:"play_count"`
		} `json:"stats"`
	} `json:"progress"`
	Show       ListItemShow `json:"show"`
	ShowId     int          `json:"show_id"`
	TotalCount int          `json:"total_count"`
}

type FetchMovieRecommendationData []ListItemMovie

func GetDynamicListMeta(id string) *dynamicListMeta {
	id = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(id, "~:"), "u:"), "/")

	if strings.Contains(id, ":") {
		parts := strings.Split(id, ":")

		meta, ok := dynamicListMetaById[parts[0]]
		if !ok {
			return nil
		}

		if meta.HasUserId {
			if len(parts) < 2 {
				return nil
			}
			meta.UserId = parts[1]
		}

		return &meta
	}

	parts := strings.Split(id, "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil
	}
	metaId := parts[0] + "/" + parts[1]
	if parts[0] == "users" && len(parts) == 3 {
		metaId = parts[2]
	}

	meta, ok := dynamicListMetaById[metaId]
	if !ok {
		return nil
	}
	if meta.HasPeriod {
		if len(parts) < 3 {
			parts[2] = "weekly"
		}
		meta.Period = parts[2]
		switch meta.Period {
		case "daily", "weekly", "monthly":
			meta.Name += " (" + strings.ToUpper(meta.Period[:1]) + meta.Period[1:] + ")"
		case "all":
			meta.Name += " (All Time)"
		default:
			return nil
		}
	}
	return &meta
}

type fetchDynamicListItemsParams struct {
	Ctx
	id string
}

func (c APIClient) fetchDynamicListItems(params *fetchDynamicListItemsParams) (APIResponse[FetchListItemsData], error) {
	meta := GetDynamicListMeta(params.id)
	if meta == nil {
		return newAPIResponse(nil, FetchListItemsData{}), errors.New("invalid id")
	}

	items := FetchListItemsData{}

	path := meta.Endpoint
	if meta.HasPeriod {
		path = strings.Replace(path, "{period}", meta.Period, 1)
	}
	if meta.HasUserId {
		path = strings.Replace(path, "{user_id}", meta.UserId, 1)
	}

	hiddenItemIdsMap := map[int]struct{}{}
	var marker string
	if meta.Endpoint == dynamicListMetaById["progress"].Endpoint {
		marker = strconv.FormatInt(time.Now().UnixMilli(), 10)

		res, err := c.FetchHiddenItems(&FetchHiddenItemsParams{
			Section: "progress_watched",
			Type:    ItemTypeShow,
		})
		if err != nil {
			response := newAPIResponse(nil, items)
			response.Header = res.Header
			response.StatusCode = res.StatusCode
			return response, err
		}
		for _, item := range res.Data {
			if item.Type == ItemTypeShow && item.Show != nil {
				hiddenItemIdsMap[item.Show.Ids.Trakt] = struct{}{}
			}
		}
	}

	hasMore := true
	limit := 100
	page := 1
	maxPage := 5
	var res *http.Response
	var err error
	for hasMore {
		log.Debug("fetching dynamic list page", "id", params.id, "page", page)

		p := &Ctx{}
		p.Query = &url.Values{}
		p.Query.Set("extended", "full,images")
		if !meta.NoPage {
			p.Query.Set("page", strconv.Itoa(page))
		}
		if !meta.NoLimit {
			p.Query.Set("limit", strconv.Itoa(limit))
		}

		p.BeforeDo(meta.BeforeRequest)

		switch meta.Endpoint {
		case dynamicListMetaById["movies/popular"].Endpoint, dynamicListMetaById["movies/recommendations"].Endpoint:
			response := listResponseData[ListItemMovie]{}
			res, err = c.Request("GET", path, p, &response)
			if err != nil {
				break
			}

			for i := range response.data {
				item := ListItem{}
				item.Type = meta.ItemType
				item.Movie = &response.data[i]
				items = append(items, item)
			}

		case dynamicListMetaById["shows/popular"].Endpoint, dynamicListMetaById["shows/recommendations"].Endpoint:
			response := listResponseData[ListItemShow]{}
			res, err = c.Request("GET", path, p, &response)
			if err != nil {
				break
			}

			for i := range response.data {
				item := ListItem{}
				item.Type = meta.ItemType
				item.Show = &response.data[i]
				items = append(items, item)
			}

		case dynamicListMetaById["progress"].Endpoint:
			p.Query.Del("extended")
			p.Query.Set("marker", marker)
			p.Query.Set("intent", "continue")

			response := listResponseData[SyncProgressUpNextNitroItem]{}
			res, err = c.Request("GET", path, p, &response)
			if err != nil {
				break
			}
			for i := range response.data {
				if _, hidden := hiddenItemIdsMap[response.data[i].ShowId]; hidden {
					continue
				}
				data := &response.data[i]
				item := ListItem{Type: ItemTypeShow}
				item.Show = &data.Show
				nextEpisode := data.Progress.NextEpisode
				if nextEpisode.EpisodeType == EpisodeTypeStandard && nextEpisode.IsAired() && nextEpisode.Season != 0 && nextEpisode.Number != 0 {
					item.NextEpisode = &listItemNextEpisode{
						Season:  data.Progress.NextEpisode.Season,
						Episode: data.Progress.NextEpisode.Number,
					}
				}
				items = append(items, item)
			}

		default:
			response := listResponseData[ListItem]{}
			res, err = c.Request("GET", path, p, &response)
			if err != nil {
				break
			}

			for i := range response.data {
				item := &response.data[i]
				if meta.ItemType != "" {
					item.Type = meta.ItemType
				}
				items = append(items, *item)
			}
		}

		if err != nil {
			break
		}

		hasMore = !meta.NoPage && page < maxPage && res.Header.Get("X-Pagination-Page") != res.Header.Get("X-Pagination-Page-Count")
		page++
	}

	return newAPIResponse(res, items), err
}
