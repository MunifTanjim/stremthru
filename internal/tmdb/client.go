package tmdb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
	"golang.org/x/oauth2"
)

type APIClientConfigOAuth struct {
	Config         oauth2.Config
	GetTokenSource func(oauth2.Config) oauth2.TokenSource
}

type APIClientConfig struct {
	HTTPClient *http.Client
	OAuth      APIClientConfigOAuth
}

type APIClientOAuth struct {
	Config oauth2.Config
	client *APIClient
}

type APIClient struct {
	BaseURL    *url.URL
	httpClient *http.Client
	OAuth      APIClientOAuth

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.DefaultHTTPClient
	}

	c := &APIClient{}

	c.BaseURL = util.MustParseURL("https://api.themoviedb.org")

	c.OAuth.Config = oauth2.Config{
		Endpoint:    conf.OAuth.Config.Endpoint,
		RedirectURL: conf.OAuth.Config.RedirectURL,
	}
	c.OAuth.client = c

	tokenSource := conf.OAuth.GetTokenSource(c.OAuth.Config)
	if tokenSource == nil {
		c.httpClient = conf.HTTPClient
	} else {
		c.httpClient = oauth2.NewClient(
			context.WithValue(context.Background(), oauth2.HTTPClient, conf.HTTPClient),
			tokenSource,
		)
	}

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Set("Accept", "application/json")
	}

	return c
}

type Ctx = request.Ctx

type ResponseError struct {
	StatusMessage string `json:"status_message"`
	StatusCode    int    `json:"status_code"`
	Success       bool   `json:"success"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseContainer interface {
	GetError() error
}

func (r *ResponseError) GetError() error {
	if r == nil || r.Success || r.StatusCode == 0 {
		return nil
	}
	return r
}

func extractResponseError(v ResponseContainer) error {
	if err := v.GetError(); err != nil {
		return err
	}
	return nil
}

func processResponseBody(res *http.Response, err error, v ResponseContainer) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return extractResponseError(v)
}

func (c APIClient) Request(method, path string, params request.Context, v ResponseContainer) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := c.httpClient.Do(req)
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

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
}

func newAPIResponse[T any](res *http.Response, data T) APIResponse[T] {
	apiResponse := APIResponse[T]{
		StatusCode: 503,
		Data:       data,
	}
	if res != nil {
		apiResponse.Header = res.Header
		apiResponse.StatusCode = res.StatusCode
	}
	return apiResponse
}
