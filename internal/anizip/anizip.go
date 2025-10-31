package anizip

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
)

type APIClientConfig struct {
	HTTPClient *http.Client
	agent      string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	agent      string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {

	if conf.HTTPClient == nil {
		conf.HTTPClient = config.DefaultHTTPClient
	}

	c := &APIClient{}

	baseUrl, err := url.Parse("https://api.ani.zip")
	if err != nil {
		panic(err)
	}

	c.BaseURL = baseUrl

	c.HTTPClient = conf.HTTPClient

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
	}

	return c
}

type Ctx = request.Ctx

func processResponseBody(res *http.Response, err error, v any) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	contentType := res.Header.Get("Content-Type")

	if res.StatusCode >= 400 {
		if !strings.Contains(contentType, "application/json") {
			err := core.NewAPIError(string(body))
			err.StatusCode = res.StatusCode
			return err
		}
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return nil
}

func (c APIClient) Request(method, path string, params request.Context, v any) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := c.HTTPClient.Do(req)
	err = processResponseBody(res, err, v)
	if err != nil {
		error := core.NewUpstreamError("")
		if rerr, ok := err.(*core.Error); ok {
			error.Msg = rerr.Msg
			error.Code = rerr.Code
			error.StatusCode = rerr.StatusCode
			error.UpstreamCause = rerr
		} else {
			error.Cause = err
		}
		error.InjectReq(req)
		return res, err
	}
	return res, nil
}

type GetMappingsData struct {
	Titles struct {
		En string `json:"en"`
	} `json:"titles"`
	Mappings struct {
		Type        string `json:"type"` // TV
		AniDB       int    `json:"anidb_id"`
		AniList     int    `json:"anilist_id"`
		AniSearch   int    `json:"anisearch_id"`
		AnimePlanet string `json:"animeplanet_id"`
		IMDB        string `json:"imdb_id"`
		Kitsu       int    `json:"kitsu_id"`
		LiveChart   int    `json:"livechart_id"`
		MAL         int    `json:"mal_id"`
		NotifyMoe   string `json:"notifymoe_id"`
		TMDB        string `json:"themoviedb_id"`
		TVDB        int    `json:"thetvdb_id"`
	} `json:"mappings"`
}

type GetMappingsParams struct {
	Ctx
	Service string
	Id      string
}

func (c APIClient) GetMappings(params *GetMappingsParams) (*GetMappingsData, error) {
	params.Query = &url.Values{}
	switch params.Service {
	case "tmdb":
		params.Service = "themoviedb"
	case "tvdb":
		params.Service = "thetvdb"
	}
	params.Query.Set(params.Service+"_id", params.Id)

	response := GetMappingsData{}
	time.Sleep(500 * time.Millisecond)
	res, err := c.Request("GET", "/mappings", params, &response)
	if err != nil || res.StatusCode != 200 {
		if res != nil && res.StatusCode == 404 {
			idInt, _ := strconv.Atoi(params.Id)
			switch params.Service {
			case "anidb":
				response.Mappings.AniDB = idInt
			case "anilist":
				response.Mappings.AniList = idInt
			case "anisearch":
				response.Mappings.AniSearch = idInt
			case "animeplanet":
				response.Mappings.AnimePlanet = params.Id
			case "imdb":
				response.Mappings.IMDB = params.Id
			case "kitsu":
				response.Mappings.Kitsu = idInt
			case "livechart":
				response.Mappings.LiveChart = idInt
			case "mal":
				response.Mappings.MAL = idInt
			case "notifymoe":
				response.Mappings.NotifyMoe = params.Id
			case "themoviedb":
				response.Mappings.TMDB = params.Id
			case "thetvdb":
				response.Mappings.TVDB = idInt
			}
			return &response, nil
		}
		return nil, errors.Join(core.NewAPIError("failed to get mappings"), err)
	}
	return &response, nil
}
