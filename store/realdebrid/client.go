package realdebrid

import (
	"log"
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

var DefaultHTTPTransport = core.DefaultHTTPTransport
var DefaultHTTPClient = core.DefaultHTTPClient

type APIClientConfig struct {
	BaseURL    string // default: https://api.real-debrid.com
	APIKey     string
	HTTPClient *http.Client
	agent      string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	apiKey     string
	agent      string
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.agent == "" {
		conf.agent = "stremthru"
	}

	if conf.BaseURL == "" {
		conf.BaseURL = "https://api.real-debrid.com"
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

	return c
}

type Ctx = store.Ctx

func (c APIClient) newRequest(method, path string, params store.RequestContext) (req *http.Request, err error) {
	if params == nil {
		params = &Ctx{}
	}

	url := c.BaseURL.JoinPath(path)

	query := url.Query()

	body, contentType, err := params.PrepareBody(method, &query)
	if err != nil {
		return nil, err
	}

	url.RawQuery = query.Encode()

	println("url: ", url.String())
	log.Printf("body: %v", body)
	req, err = http.NewRequestWithContext(params.GetContext(), method, url.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+params.GetAPIKey(c.apiKey))
	req.Header.Add("User-Agent", c.agent)
	if len(contentType) > 0 {
		req.Header.Add("Content-Type", contentType)
	}

	return req, nil
}

func (c APIClient) Request(method, path string, params store.RequestContext, v ResponseContainer) (*http.Response, error) {
	req, err := c.newRequest(method, path, params)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameRealDebrid)
		error.Cause = err
		return nil, error
	}
	res, err := c.HTTPClient.Do(req)
	err = processResponseBody(res, err, v)
	if err != nil {
		return res, UpstreamErrorFromRequest(err, req, res)
	}
	return res, nil
}
