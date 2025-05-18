package mdblist

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
)

var DefaultHTTPClient = func() *http.Client {
	transport := config.DefaultHTTPTransport.Clone()
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}()

type APIClientConfig struct {
	BaseURL    string // default: https://api.mdblist.com
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	apiKey     string
	agent      string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)

	checkMagnetRetryAfter *time.Time
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.UserAgent == "" {
		conf.UserAgent = "stremthru"
	}

	if conf.BaseURL == "" {
		conf.BaseURL = "https://api.mdblist.com"
	}

	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &APIClient{}

	baseUrl, err := url.Parse(conf.BaseURL)
	if err != nil {
		panic(err)
	}

	c.BaseURL = baseUrl
	c.HTTPClient = conf.HTTPClient
	c.apiKey = conf.APIKey
	c.agent = conf.UserAgent

	c.reqQuery = func(query *url.Values, params request.Context) {
		query.Set("apikey", params.GetAPIKey(c.apiKey))
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Add("User-Agent", c.agent)
	}

	return c
}

type Ctx = request.Ctx

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := c.HTTPClient.Do(req)
	err = processResponseBody(res, err, v)
	if err != nil {
		err := UpstreamErrorWithCause(err)
		err.InjectReq(req)
		if res != nil {
			err.StatusCode = res.StatusCode
		}
		return res, err
	}
	return res, nil
}

type ResponseEnvelop interface {
	HasError() bool
	GetError() *ResponseContainer
}

type ResponseContainer struct {
	Err string `json:"error"`
}

func (e *ResponseContainer) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (r *ResponseContainer) HasError() bool {
	return r.Err != ""
}

func (r *ResponseContainer) GetError() *ResponseContainer {
	if r.HasError() {
		return r
	}
	return nil
}

func extractResponseError(v ResponseEnvelop) error {
	if v.HasError() {
		return v.GetError()
	}
	return nil
}

func processResponseBody(res *http.Response, err error, v ResponseEnvelop) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		contentType := res.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return &ResponseContainer{
				Err: string(core.ErrorCodeInternalServerError),
			}
		}
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return extractResponseError(v)
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
