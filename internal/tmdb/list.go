package tmdb

import (
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/golang-jwt/jwt/v5"
)

type PaginatedResult[T any] struct {
	Page         int `json:"page"`
	Results      []T `json:"results"`
	TotalPages   int `json:"total_pages"`
	TotalResults int `json:"total_results"`
}

type listItemCommon struct {
	Adult            bool    `json:"adult"`
	BackdropPath     string  `json:"backdrop_path"`
	GenreIds         []int   `json:"genre_ids"`
	Id               int     `json:"id"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	Popularity       float64 `json:"popularity"`
	PosterPath       string  `json:"poster_path"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
}

type ListItemMovie struct {
	listItemCommon
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Title         string `json:"title"`
	Video         bool   `json:"video"`
}

func (li *ListItemMovie) GetReleaseDate() time.Time {
	if li.ReleaseDate == "" {
		return time.Time{}
	}
	return util.MustParseTime(time.DateOnly, li.ReleaseDate)
}

type ListItemShow struct {
	listItemCommon
	OriginCountry []string `json:"origin_country"`
	OriginalName  string   `json:"original_name"`
	FirstAirDate  string   `json:"first_air_date"`
	Name          string   `json:"name"`
}

func (li *ListItemShow) GetFirstAirDate() time.Time {
	if li.FirstAirDate == "" {
		return time.Time{}
	}
	return util.MustParseTime(time.DateOnly, li.FirstAirDate)
}

type AccountRating struct {
	CreatedAt string `json:"created_at"`
	Value     int    `json:"value"`
}

type ListItemRatedMovie struct {
	ListItemMovie
	AccountRating AccountRating `json:"account_rating"`
}

type ListItemRatedShow struct {
	ListItemShow
	AccountRating AccountRating `json:"account_rating"`
}

type listItem interface {
	MediaType() MediaType
}

func (ListItemMovie) MediaType() MediaType { return MediaTypeMovie }
func (ListItemShow) MediaType() MediaType  { return MediaTypeTVShow }

type ListItem struct {
	MediaType MediaType `json:"media_type"`
	data      listItem
}

func (li *ListItem) Movie() *ListItemMovie {
	return li.data.(*ListItemMovie)
}
func (li *ListItem) Show() *ListItemShow {
	return li.data.(*ListItemShow)
}

func (li *ListItem) UnmarshalJSON(data []byte) error {
	var probe struct {
		MediaType MediaType `json:"media_type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}

	li.MediaType = probe.MediaType
	switch probe.MediaType {
	case "tv":
		var tv ListItemShow
		if err := json.Unmarshal(data, &tv); err != nil {
			return err
		}
		li.data = &tv
	case "movie":
		var movie ListItemMovie
		if err := json.Unmarshal(data, &movie); err != nil {
			return err
		}
		li.data = &movie
	default:
		return errors.New("unknown media_type: " + string(probe.MediaType))
	}
	return nil
}

type List struct {
	PaginatedResult[ListItem]
	AverageRating float64        `json:"average_rating"`
	BackdropPath  string         `json:"backdrop_path"`
	Comments      map[string]any `json:"comments"`
	CreatedBy     struct {
		AvatarPath   string `json:"avatar_path"`
		GravatarHash string `json:"gravatar_hash"`
		Id           string `json:"id"`
		Name         string `json:"name"`
		Username     string `json:"username"`
	} `json:"created_by"`
	Description string         `json:"description"`
	Id          int            `json:"id"`
	ISO31661    string         `json:"iso_3166_1"`
	ISO6391     string         `json:"iso_639_1"`
	ItemCount   int            `json:"item_count"`
	Name        string         `json:"name"`
	ObjectIds   map[string]any `json:"object_ids"`
	PosterPath  string         `json:"poster_path"`
	Public      bool           `json:"public"`
	Revenue     int            `json:"revenue"`
	Runtime     int            `json:"runtime"`
	SortBy      string         `json:"sort_by"`
}

type FetchListData struct {
	ResponseError
	List
}

type FetchListParams struct {
	Ctx
	ListId   int
	Language string
	Page     int
}

func (c APIClient) FetchList(params *FetchListParams) (APIResponse[List], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	params.Query = &query

	response := FetchListData{}
	res, err := c.Request("GET", "/4/list/"+strconv.Itoa(params.ListId), params, &response)
	return newAPIResponse(res, response.List), err
}

type fetchDynamicListDataResult struct {
	ListItem
}

type fetchListParams struct {
	Ctx
	Language string
	Page     int
	SortBy   string
}

type fetchListData[T any] struct {
	ResponseError
	PaginatedResult[T]
}

type FetchFavoriteMoviesData = fetchListData[ListItemMovie]

type FetchFavoriteMoviesParams = fetchListParams

func (c APIClient) FetchFavoriteMovies(params *FetchFavoriteMoviesParams) (APIResponse[FetchFavoriteMoviesData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchFavoriteMoviesData{}
	res, err := c.Request("GET", "/3/account/0/favorite/movies", params, &response)
	return newAPIResponse(res, response), err
}

type FetchFavoriteShowsData = fetchListData[ListItemShow]

type FetchFavoriteShowsParams = fetchListParams

func (c APIClient) FetchFavoriteShows(params *FetchFavoriteShowsParams) (APIResponse[FetchFavoriteShowsData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchFavoriteShowsData{}
	res, err := c.Request("GET", "/3/account/0/favorite/tv", params, &response)
	return newAPIResponse(res, response), err
}

type FetchRatedMoviesData = fetchListData[ListItemRatedMovie]

type FetchRatedMoviesParams = fetchListParams

func (c APIClient) FetchRatedMovies(params *FetchRatedMoviesParams) (APIResponse[FetchRatedMoviesData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchRatedMoviesData{}
	res, err := c.Request("GET", "/3/account/0/rated/movies", params, &response)
	return newAPIResponse(res, response), err
}

type FetchRatedShowsData = fetchListData[ListItemRatedShow]

type FetchRatedShowsParams = fetchListParams

func (c APIClient) FetchRatedShows(params *FetchRatedShowsParams) (APIResponse[FetchRatedShowsData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchRatedShowsData{}
	res, err := c.Request("GET", "/3/account/0/rated/tv", params, &response)
	return newAPIResponse(res, response), err
}

type FetchWatchlistMoviesData = fetchListData[ListItemMovie]

type FetchWatchlistMoviesParams = fetchListParams

func (c APIClient) FetchWatchlistMovies(params *FetchWatchlistMoviesParams) (APIResponse[FetchWatchlistMoviesData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchWatchlistMoviesData{}
	res, err := c.Request("GET", "/3/account/0/watchlist/movies", params, &response)
	return newAPIResponse(res, response), err
}

type FetchWatchlistShowsData = fetchListData[ListItemShow]

type FetchWatchlistShowsParams = fetchListParams

func (c APIClient) FetchWatchlistShows(params *FetchWatchlistShowsParams) (APIResponse[FetchWatchlistShowsData], error) {
	query := url.Values{}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	params.Query = &query

	response := FetchWatchlistShowsData{}
	res, err := c.Request("GET", "/3/account/0/watchlist/tv", params, &response)
	return newAPIResponse(res, response), err
}

type dynamicListMeta struct {
	Endpoint  string
	Name      string
	MediaType MediaType
	SortBy    string
	Public    bool

	NeedAccountObjectId bool
	IsV4                bool
}

func (m dynamicListMeta) fetchPage(client *APIClient, page int, pageSize int) (items []ListItem, totalPages, totalResults int, err error) {
	log.Debug("fetching dynamic list page", "name", m.Name, "page", page)
	params := &Ctx{}
	params.Query = &url.Values{
		"page": []string{strconv.Itoa(page)},
	}
	if m.SortBy != "" {
		params.Query.Set("sort_by", m.SortBy)
	}

	if m.IsV4 {
		response := fetchListData[ListItem]{}
		endpoint := m.Endpoint
		if m.NeedAccountObjectId {
			if token, err := client.OAuth.TokenSource.Token(); err != nil {
				return nil, totalPages, totalResults, err
			} else if t, _, err := jwt.NewParser().ParseUnverified(token.AccessToken, jwt.MapClaims{}); err != nil {
				return nil, totalPages, totalResults, err
			} else if sub, err := t.Claims.GetSubject(); err != nil {
				return nil, totalPages, totalResults, err
			} else {
				endpoint = strings.ReplaceAll(endpoint, "{account_object_id}", sub)
			}
		}
		_, err := client.Request("GET", endpoint, params, &response)
		if err != nil {
			return nil, totalPages, totalResults, err
		}
		totalPages, totalResults = response.TotalPages, response.TotalResults
		return response.Results, totalPages, totalResults, nil
	}

	items = make([]ListItem, 0, pageSize)
	switch m.MediaType {
	case MediaTypeMovie:
		response := fetchListData[ListItemMovie]{}
		_, err := client.Request("GET", m.Endpoint, params, &response)
		if err != nil {
			return nil, totalPages, totalResults, err
		}
		totalPages, totalResults = response.TotalPages, response.TotalResults
		for i := range response.Results {
			items = append(items, ListItem{
				MediaType: m.MediaType,
				data:      &response.Results[i],
			})
		}
	case MediaTypeTVShow:
		response := fetchListData[ListItemShow]{}
		_, err := client.Request("GET", m.Endpoint, params, &response)
		if err != nil {
			return nil, totalPages, totalResults, err
		}
		totalPages, totalResults = response.TotalPages, response.TotalResults
		for i := range response.Results {
			items = append(items, ListItem{
				MediaType: m.MediaType,
				data:      &response.Results[i],
			})
		}
	}
	return items, totalPages, totalResults, nil
}

func (m dynamicListMeta) Fetch(client *APIClient) (*List, error) {
	l := List{
		Name:      m.Name,
		Public:    m.Public,
		ItemCount: 0,
	}

	firstPageItems, totalPages, _, err := m.fetchPage(client, 1, 0)
	if err != nil {
		return nil, err
	}

	pageSize := len(firstPageItems)
	maxPage := min(25, totalPages)

	var mutex sync.Mutex
	l.Results = make([]ListItem, maxPage*pageSize)
	copy(l.Results, firstPageItems)

	if maxPage == 1 {
		l.ItemCount = len(l.Results)
		return &l, nil
	}

	pageNumbers := util.IntRange(2, maxPage)
	for cPageNumbers := range slices.Chunk(pageNumbers, 4) {
		var wg sync.WaitGroup
		var wgErr error
		for _, page := range cPageNumbers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				items, _, _, err := m.fetchPage(client, page, pageSize)
				mutex.Lock()
				defer mutex.Unlock()
				if err != nil {
					log.Error("failed to fetch dynamic list page", "error", err, "name", m.Name, "page", page)
					wgErr = errors.Join(wgErr, err)
					return
				}
				startIndex := (page - 1) * pageSize
				for i := range items {
					l.Results[startIndex+i] = items[i]
				}
			}()
		}
		wg.Wait()
		if wgErr != nil {
			return nil, wgErr
		}
	}

	l.ItemCount = len(l.Results)
	return &l, nil
}

var dynamicListMetaById = map[string]dynamicListMeta{
	"favorites/movie": {
		Endpoint:  "/3/account/0/favorite/movies",
		Name:      "Favorites",
		MediaType: MediaTypeMovie,
		SortBy:    "created_at.desc",
	},
	"favorites/tv": {
		Endpoint:  "/3/account/0/favorite/tv",
		Name:      "Favorites",
		MediaType: MediaTypeTVShow,
		SortBy:    "created_at.desc",
	},
	"ratings/movie": {
		Endpoint:  "/3/account/0/rated/movies",
		Name:      "Ratings",
		MediaType: MediaTypeMovie,
		SortBy:    "created_at.desc",
	},
	"ratings/tv": {
		Endpoint:  "/3/account/0/rated/tv",
		Name:      "Ratings",
		MediaType: MediaTypeTVShow,
		SortBy:    "created_at.desc",
	},
	"recommendations/movie": {
		Endpoint:            "/4/account/{account_object_id}/movie/recommendations",
		Name:                "Recommendations",
		MediaType:           MediaTypeMovie,
		NeedAccountObjectId: true,
		IsV4:                true,
	},
	"recommendations/tv": {
		Endpoint:            "/4/account/{account_object_id}/tv/recommendations",
		Name:                "Recommendations",
		MediaType:           MediaTypeTVShow,
		NeedAccountObjectId: true,
		IsV4:                true,
	},
	"watchlist/movie": {
		Endpoint:  "/3/account/0/watchlist/movies",
		Name:      "Watchlist",
		MediaType: MediaTypeMovie,
		SortBy:    "created_at.desc",
	},
	"watchlist/tv": {
		Endpoint:  "/3/account/0/watchlist/tv",
		Name:      "Watchlist",
		MediaType: MediaTypeTVShow,
		SortBy:    "created_at.desc",
	},
	"movie": {
		Endpoint:  "/3/movie/popular",
		Name:      "Popular",
		MediaType: MediaTypeMovie,
		Public:    true,
	},
	"movie/now-playing": {
		Endpoint:  "/3/movie/now_playing",
		Name:      "Now Playing",
		MediaType: MediaTypeMovie,
		Public:    true,
	},
	"movie/upcoming": {
		Endpoint:  "/3/movie/upcoming",
		Name:      "Upcoming",
		MediaType: MediaTypeMovie,
		Public:    true,
	},
	"movie/top-rated": {
		Endpoint:  "/3/movie/top_rated",
		Name:      "Top Rated",
		MediaType: MediaTypeMovie,
		Public:    true,
	},
	"tv": {
		Endpoint:  "/3/tv/popular",
		Name:      "Popular",
		MediaType: MediaTypeTVShow,
		Public:    true,
	},
	"tv/airing-today": {
		Endpoint:  "/3/tv/airing_today",
		Name:      "Airing Today",
		MediaType: MediaTypeTVShow,
		Public:    true,
	},
	"tv/on-the-air": {
		Endpoint:  "/3/tv/on_the_air",
		Name:      "Currently Airing",
		MediaType: MediaTypeTVShow,
		Public:    true,
	},
	"tv/top-rated": {
		Endpoint:  "/3/tv/top_rated",
		Name:      "Top Rated",
		MediaType: MediaTypeTVShow,
		Public:    true,
	},
}

func GetDynamicListMeta(id string) *dynamicListMeta {
	id = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(id, "~:"), "u:"), "/")
	if meta, ok := dynamicListMetaById[id]; ok {
		return &meta
	}
	return nil
}
