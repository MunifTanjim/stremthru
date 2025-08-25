package tmdb

import (
	"net/url"
	"strings"
)

type GetMovieExternalIdsData struct {
	ResponseError
	Id          int    `json:"id"`
	IMDBId      string `json:"imdb_id"`
	WikidataId  string `json:"wikidata_id"`
	FacebookId  string `json:"facebook_id"`
	InstagramId string `json:"instagram_id"`
	TwitterId   string `json:"twitter_id"`
}

type GetMovieExternalIdsParams struct {
	Ctx
	MovieId string
}

func (c APIClient) GetMovieExternalIds(params *GetMovieExternalIdsParams) (APIResponse[GetMovieExternalIdsData], error) {
	var response GetMovieExternalIdsData
	res, err := c.Request("GET", "/3/movie/"+params.MovieId+"/external_ids", &params.Ctx, &response)
	return newAPIResponse(res, response), err
}

type GetTVExternalIdsData struct {
	ResponseError
	Id          int    `json:"id"`
	IMDBId      string `json:"imdb_id"`
	FreebaseMId string `json:"freebase_mid"`
	FreebaseId  string `json:"freebase_id"`
	TVDBId      int    `json:"tvdb_id"`
	TVRageId    int    `json:"tvrage_id"`
	WikidataId  string `json:"wikidata_id"`
	FacebookId  string `json:"facebook_id"`
	InstagramId string `json:"instagram_id"`
	TwitterId   string `json:"twitter_id"`
}

type GetTVExternalIdsParams struct {
	Ctx
	SeriesId string
}

func (c APIClient) GetTVExternalIds(params *GetTVExternalIdsParams) (APIResponse[GetTVExternalIdsData], error) {
	var response GetTVExternalIdsData
	res, err := c.Request("GET", "/3/tv/"+params.SeriesId+"/external_ids", &params.Ctx, &response)
	return newAPIResponse(res, response), err
}

type FindByIdData struct {
	ResponseError
	MovieResults []ListItemMovie `json:"movie_results"`
	TVResults    []ListItemShow  `json:"tv_results"`
}

func (data *FindByIdData) Movie() *ListItemMovie {
	if len(data.MovieResults) == 0 {
		return nil
	}
	return &data.MovieResults[0]
}

func (data *FindByIdData) Show() *ListItemShow {
	if len(data.TVResults) == 0 {
		return nil
	}
	return &data.TVResults[0]
}

type FindByIdParams struct {
	Ctx
	ExternalId     string
	ExternalSource string // imdb_id / tvdb_id
	Language       string
}

func (c APIClient) FindById(params *FindByIdParams) (APIResponse[FindByIdData], error) {
	query := url.Values{}
	if params.ExternalSource == "" && strings.HasPrefix(params.ExternalId, "tt") {
		params.ExternalSource = "imdb_id"
	}
	query.Set("external_source", params.ExternalSource)
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	params.Query = &query

	response := FindByIdData{}
	res, err := c.Request("GET", "/3/find/"+params.ExternalId, params, &response)
	return newAPIResponse(res, response), err
}
