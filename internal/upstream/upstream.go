package upstream

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

var DefaultHTTPTransport = func() *http.Transport {
	transport := request.DefaultHTTPTransport.Clone()
	transport.Proxy = nil
	return transport
}()
var DefaultHTTPClient = func() *http.Client {
	return &http.Client{
		Transport: DefaultHTTPTransport,
	}
}()

type APIClientConfig struct {
	BaseURL    string
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
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.agent == "" {
		conf.agent = "stremthru"
	}

	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &APIClient{}

	if conf.BaseURL != "" {
		baseUrl, err := url.Parse(conf.BaseURL)
		if err != nil {
			panic(err)
		}
		c.BaseURL = baseUrl
	}

	c.HTTPClient = conf.HTTPClient
	c.apiKey = conf.APIKey
	c.agent = conf.agent

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Set("X-StremThru-Upstream-Token", params.GetAPIKey(c.apiKey))
		header.Add("User-Agent", c.agent)
	}

	return c
}

type Ctx = request.Ctx

type ResponseEnvelop interface {
	GetError() error
}

type ResponseError struct {
	Code       core.ErrorCode `json:"code"`
	Message    string         `json:"message"`
	StatusCode int            `json:"status_code"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type Response[D interface{}] struct {
	Data  D              `json:"data,omitempty"`
	Error *ResponseError `json:"error,omitempty"`
}

func (r Response[any]) GetError() error {
	if r.Error == nil {
		return nil
	}
	return r.Error
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

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return v.GetError()
}

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
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

type CheckMagnetParams struct {
	store.CheckMagnetParams
	StoreName  store.StoreName
	StoreToken string
}

func (c APIClient) CheckMagnet(params *CheckMagnetParams) (request.APIResponse[store.CheckMagnetData], error) {
	params.Form = &url.Values{"magnet": params.Magnets}
	params.Headers = &http.Header{
		"X-StremThru-Store-Name":          []string{string(params.StoreName)},
		"X-StremThru-Store-Authorization": []string{"Bearer " + params.StoreToken},
	}

	response := &Response[store.CheckMagnetData]{}
	res, err := c.Request("GET", "/v0/store/magnets/check", params, response)
	return request.NewAPIResponse(res, response.Data), err
}

type TrackMagnetParams struct {
	store.Ctx
	StoreName  store.StoreName    `json:"-"`
	StoreToken string             `json:"-"`
	Hash       string             `json:"hash"`
	Files      []store.MagnetFile `json:"files"`
	IsMiss     bool               `json:"is_miss"`
}

type TrackMagnetData struct{}

func (c APIClient) TrackMagnet(params *TrackMagnetParams) (request.APIResponse[TrackMagnetData], error) {
	params.Headers = &http.Header{
		"X-StremThru-Store-Name":          []string{string(params.StoreName)},
		"X-StremThru-Store-Authorization": []string{"Bearer " + params.StoreToken},
	}
	params.JSON = params

	response := &Response[TrackMagnetData]{}
	res, err := c.Request("POST", "/v0/store/magnets/check", params, response)
	return request.NewAPIResponse(res, response.Data), err
}
