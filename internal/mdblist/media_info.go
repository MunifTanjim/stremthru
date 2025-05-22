package mdblist

import (
	"encoding/json"

	"github.com/MunifTanjim/stremthru/core"
)

type MediaInfo struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Year         int    `json:"year"`
	Released     string `json:"released"`
	Description  string `json:"description"`
	Runtime      int    `json:"runtime"`
	Score        int    `json:"score"`
	ScoreAverage int    `json:"score_average"`
	Ids          struct {
		IMDB  string `json:"imdb"`
		Trakt int    `json:"trakt"`
		TMDB  int    `json:"tmdb"`
		TVDB  int    `json:"tvdb"`
		MAL   int    `json:"mal"`
	}
	Type    string `json:"type"` // movie / show
	Ratings []struct {
		Source string  `json:"source"` // imdb / metacritic / metacriticuser / trakt / tomatoes / popcorn / tmdb / letterboxd / rogerebert / myanimelist
		Value  float32 `json:"value,omitempty"`
		Score  float32 `json:"score,omitempty"`
		Votes  int     `json:"votes,omitempty"`
		Url    any     `json:"url,omitempty"` // int or string
	} `json:"ratings"`
	Streams []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"streams"`
	WatchProviders []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"watch_providers"`
	Language       string `json:"language"`
	SpokenLanguage string `json:"spoken_language"`
	Country        string `json:"country"`
	Certification  string `json:"certification"`
	Commonsense    bool   `json:"commonsense"`
	AgeRating      int    `json:"age_rating"`
	Status         string `json:"status"` // released
	Trailer        string `json:"trailer,omitempty"`
	Poster         string `json:"poster"`
	Backdrop       string `json:"backdrop"`
	Reviews        []struct {
		UpdatedAt  string `json:"updated_at"`
		Author     string `json:"author"`
		Rating     int    `json:"rating"`
		ProviderId int    `json:"provider_id"`
		Content    string `json:"content"`
	} `json:"reviews,omitempty"`
	Keywords []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"keywords,omitempty"`
	ProductionCompanies []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"production_companies"`
	TVDBId  int `json:"tvdbid"`
	Seasons []struct {
		TMDBId       int    `json:"tmdbid"`
		Name         string `json:"name"`
		AirDate      string `json:"air_date"`
		EpisodeCount int    `json:"episode_count"`
		SeasonNumber int    `json:"season_number"`
		TomatoFresh  any    `json:"tomatofresh"`
		PosterPath   string `json:"poster_path"`
	}
	Genres []struct {
		Id    int    `json:"id"`
		Title string `json:"title"`
	} `json:"genres"`
}

type GetMediaInfoBatchData []MediaInfo

type getMediaInfoBatchData struct {
	ResponseContainer
	data GetMediaInfoBatchData
}

func (d *getMediaInfoBatchData) UnmarshalJSON(data []byte) (err error) {
	var rerr ResponseContainer
	if err = json.Unmarshal(data, &rerr); err == nil {
		d.ResponseContainer = rerr
		return nil
	}

	var items GetMediaInfoBatchData
	if err = json.Unmarshal(data, &items); err == nil {
		d.data = items
		return nil
	}

	apiErr := core.NewAPIError("failed to parse response")
	apiErr.Cause = err
	return apiErr
}

type GetMediaInfoBatchParams struct {
	Ctx
	MediaProvider    string   `json:"-"` // `tmdb` / `imdb` / `trakt` / `tmdb` / `tvdb` / `mal`
	MediaType        string   `json:"-"` // `movie` / `show` / `any`
	Ids              []string `json:"ids"`
	AppendToResponse []string `json:"append_to_response,omitempty"` // `review` / `keywords`
}

func (c APIClient) GetMediaInfoBatch(params *GetMediaInfoBatchParams) (APIResponse[GetMediaInfoBatchData], error) {
	if len(params.Ids) > 200 {
		return newAPIResponse(nil, GetMediaInfoBatchData{}), core.NewAPIError("ids length exceeds 200")
	}
	params.JSON = params
	response := &getMediaInfoBatchData{}
	res, err := c.Request("POST", "/"+params.MediaProvider+"/"+params.MediaType, params, response)
	return newAPIResponse(res, response.data), err
}
