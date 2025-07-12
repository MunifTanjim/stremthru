package tmdb

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
	res, err := c.Request("GET", "/3/movie/"+params.MovieId+"/external_ids", params.Ctx, &response)
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
	res, err := c.Request("GET", "/3/tv/"+params.SeriesId+"/external_ids", params.Ctx, &response)
	return newAPIResponse(res, response), err
}
