package jackett

import (
	"net/http"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type ClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	UserAgent  string
}

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client

	userAgent string
	apiKey    string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)

	torznabClientById cache.Cache[TorznabClient]
}

func NewClient(conf *ClientConfig) *Client {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	}

	if conf.UserAgent == "" {
		conf.UserAgent = "stremthru/" + config.Version
	}

	c := Client{
		HTTPClient: conf.HTTPClient,
		userAgent:  conf.UserAgent,
		apiKey:     conf.APIKey,
	}

	c.BaseURL = util.MustParseURL(conf.BaseURL)

	c.reqQuery = func(query *url.Values, params request.Context) {
		query.Set("apikey", c.apiKey)
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Set("User-Agent", c.userAgent)
	}

	c.torznabClientById = cache.NewCache[TorznabClient](&cache.CacheConfig{
		Lifetime: 30 * time.Minute,
		Name:     "torznab:jackett:torznab-client",
	})

	return &c
}

type Ctx = request.Ctx

func (c *Client) Request(method, path string, params request.Context, v request.ResponseContainer) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := params.DoRequest(c.HTTPClient, req)
	err = request.ProcessResponseBody(res, err, v)
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
