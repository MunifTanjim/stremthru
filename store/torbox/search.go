package torbox

import (
	"net/url"
	"strconv"
)

type UsenetSearchByIDDataMetadata struct {
	GlobalID   string   `json:"globalID"`
	ID         string   `json:"id"`
	IMDBID     string   `json:"imdb_id"`
	TMDBID     string   `json:"tmdb_id"`
	TVDBID     string   `json:"tvdb_id"`
	TVMazeID   string   `json:"tvmaze_id"`
	TraktID    string   `json:"trakt_id"`
	MALID      string   `json:"mal_id"`
	AniListID  string   `json:"anilist_id"`
	KitsuID    string   `json:"kitsu_id"`
	SimklID    string   `json:"simkl_id"`
	Title      string   `json:"title"`
	Titles     []string `json:"titles"`
	TitlesFull []struct {
		Language string `json:"language"` // EN
		Title    string `json:"title"`
	} `json:"titles_full"`
	TranslatedTitles []any    `json:"translated_titles"`
	Link             any      `json:"link"`
	Description      string   `json:"description"`
	Genres           []string `json:"genres"`
	MediaType        string   `json:"mediaType"` // movie
	Rating           float32  `json:"rating"`
	Popularity       int      `json:"popularity"`
	Languages        []string `json:"languages"`
	ContentRating    string   `json:"contentRating"`
	Actors           []any    `json:"actors"`
	Trailer          any      `json:"trailer"`
	Characters       []any    `json:"characters"`
	Image            string   `json:"image"`
	IsAdult          bool     `json:"isAdult"`
	Type             string   `json:"type"` // movie
	ReleasedDate     string   `json:"releasedDate"`
	SeasonsNumber    int      `json:"seasonsNumber"`
	EpisodesNumber   int      `json:"episodesNumber"`
	Runtime          string   `json:"runtime"` // 2h 7m
	ReleaseYears     int      `json:"releaseYears"`
	Keywords         []string `json:"keywords"`
	Backdrop         string   `json:"backdrop"`
}

type UsenetSearchByIDDataNZB struct {
	Hash     string `json:"hash"`
	RawTitle string `json:"raw_title"`
	Title    string `json:"title"`
	// TitleParsedData struct {
	// 	Resolution string `json:"resolution"`
	// 	Year       int    `json:"year"`
	// 	Codec      string `json:"codec"`
	// 	Audio      string `json:"audio"`
	// 	Remux      bool   `json:"remux"`
	// 	Title      string `json:"title"`
	// 	Excess     string `json:"excess"` // or []string
	// 	Encoder    string `json:"encoder"`
	// 	Language   string `json:"language"`
	// 	Site       string `json:"site"`
	// 	HDR        bool   `json:"hdr"`
	// } `json:"title_parsed_data"`
	Size       int64  `json:"size"`
	Tracker    string `json:"tracker"`
	Categories []int  `json:"categories"`
	Files      int    `json:"files"`
	NZB        string `json:"nzb"`
	Age        string `json:"age"`
	Type       string `json:"type"` // "usenet"
	UserSearch bool   `json:"user_search"`
	Cached     bool   `json:"cached"`
	Owned      bool   `json:"owned"`
}

type UsenetSearchByIDData struct {
	Metadata  *UsenetSearchByIDDataMetadata `json:"metadata,omitempty"`
	NZBs      []UsenetSearchByIDDataNZB     `json:"nzbs"`
	TimeTaken float32                       `json:"time_taken"`
	Cache     bool                          `json:"cache"`
	TotalNZBs int                           `json:"total_nzbs"`
}

type SearchUsenetByIDParams struct {
	Ctx
	Metadata          bool
	Season            int
	Episode           int
	CheckCache        bool
	CheckOwned        bool
	SearchUserEngines bool
	IDType            string // imdb
	ID                string
}

func (c APIClient) SearchUsenetByID(params *SearchUsenetByIDParams) (APIResponse[UsenetSearchByIDData], error) {
	query := &url.Values{}
	if params.Metadata {
		query.Add("metadata", "true")
	}
	if params.Season != 0 {
		query.Add("season", strconv.Itoa(params.Season))
	}
	if params.Episode != 0 {
		query.Add("episode", strconv.Itoa(params.Episode))
	}
	if params.CheckCache {
		query.Add("check_cache", "true")
	}
	if params.CheckOwned {
		query.Add("check_owned", "true")
	}
	if params.SearchUserEngines {
		query.Add("search_user_engines", "true")
	}

	if params.IDType == "" {
		params.IDType = "imdb"
	}

	params.Query = query
	response := &Response[UsenetSearchByIDData]{}
	res, err := c.RequestSearch("GET", "/usenet/"+params.IDType+":"+params.ID, params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}
