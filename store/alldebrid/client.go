package alldebrid

import (
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

var DefaultHTTPTransport = core.DefaultHTTPTransport
var DefaultHTTPClient = core.DefaultHTTPClient

type APIClientConfig struct {
	BaseURL    string // default: https://api.alldebrid.com
	APIKey     string
	HTTPClient *http.Client
	agent      string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	apiKey     string
	agent      string
	reqQuery   func(query *url.Values, params store.RequestContext)
	reqHeader  func(query *http.Header, params store.RequestContext)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.agent == "" {
		conf.agent = "stremthru"
	}

	if conf.BaseURL == "" {
		conf.BaseURL = "https://api.alldebrid.com"
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
	c.agent = conf.agent

	c.reqQuery = func(query *url.Values, params store.RequestContext) {
		query.Set("agent", c.agent)
	}

	c.reqHeader = func(header *http.Header, params store.RequestContext) {
		header.Set("Authorization", "Bearer "+params.GetAPIKey(c.apiKey))
		header.Add("User-Agent", c.agent)
	}

	return c
}

type Ctx = store.Ctx

func (c APIClient) Request(method, path string, params store.RequestContext, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameAlldebrid)
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
