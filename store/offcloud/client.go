package offcloud

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

var DefaultHTTPTransport = request.DefaultHTTPTransport
var DefaultHTTPClient = request.DefaultHTTPClient

type APIClientConfig struct {
	BaseURL    string // default: https://offcloud.com
	APIKey     string
	HTTPClient *http.Client
	agent      string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	apiKey     string
	agent      string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)

	authCache cache.Cache[cachedAuth]
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.agent == "" {
		conf.agent = "stremthru"
	}

	if conf.BaseURL == "" {
		conf.BaseURL = "https://offcloud.com"
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

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Add("User-Agent", c.agent)
	}

	c.authCache = cache.NewCache[cachedAuth](&cache.CacheConfig{
		Name:     "store:offcloud:cookie",
		Lifetime: 6 * time.Hour,
	})

	return c
}

type Ctx = request.Ctx

func (c APIClient) doRequest(req *http.Request, v ResponseEnvelop) (*http.Response, error) {
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

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameOffcloud)
		error.Cause = err
		return nil, error
	}
	return c.doRequest(req, v)
}

func (c APIClient) ServerRequest(server, method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	baseUrl := c.BaseURL.JoinPath(path)
	baseUrl.Host = server + "." + baseUrl.Host
	req, err := params.NewRequest(baseUrl, method, "", c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameOffcloud)
		error.Cause = err
		return nil, error
	}
	return c.doRequest(req, v)
}

type GetFileSizeParams struct {
	Ctx
	Link      string // if `Link` is present, other fields are not needed
	Server    string
	RequestId string
	Index     int
	FileName  string
}

type GetFileSizeData struct {
	ResponseContainer
}

func (c APIClient) GetFileSize(params *GetFileSizeParams) (APIResponse[int], error) {
	size := -1

	server := ""
	path := ""
	if params.Link == "" {
		server = params.Server
		path = "/cloud/download/" + params.RequestId + "/" + strconv.Itoa(params.Index) + "/" + params.FileName
	} else {
		info, err := CloudDownloadLink(params.Link).parse()
		if err != nil {
			return newAPIResponse(nil, size), err
		}
		server = info.server
		path = info.path
	}

	response := &GetFileSizeData{}
	res, err := c.ServerRequest(server, "HEAD", path, params, response)
	if s, err := strconv.Atoi(res.Header.Get("Content-Length")); err == nil {
		size = s
	}
	return newAPIResponse(res, size), err
}
