package stremio_api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
)

var DefaultHTTPTransport = func() *http.Transport {
	transport := request.DefaultHTTPTransport.Clone()
	return transport
}()
var DefaultHTTPClient = func() *http.Client {
	return &http.Client{
		Transport: DefaultHTTPTransport,
	}
}()

type ClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewClient(conf *ClientConfig) *Client {
	if conf.BaseURL == "" {
		conf.BaseURL = "https://api.strem.io"
	}

	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &Client{}

	baseUrl, err := url.Parse(conf.BaseURL)
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

type ResponseEnvelop interface {
	GetError() error
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type Response[D interface{}] struct {
	Result D              `json:"result,omitempty"`
	Error  *ResponseError `json:"error,omitempty"`
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

func (c Client) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
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

type LoginParams struct {
	Ctx
	Email    string `json:"email"`
	Password string `json:"password"`
	Facebook bool   `json:"facebook"`
	Type     string `json:"type"`
}

func (c Client) Login(params *LoginParams) (request.APIResponse[LoginData], error) {
	params.Facebook = false
	params.Type = "login"
	params.JSON = params

	response := &Response[LoginData]{}
	res, err := c.Request("POST", "/api/login", params, response)
	return request.NewAPIResponse(res, response.Result), err

}

type requestPayload struct {
	AuthKey string `json:"authKey"`
	Type    string `json:"type"`
}

type getUserPayload struct {
	requestPayload
}

type GetUserParams struct {
	Ctx
}

func (c Client) GetUser(params *GetUserParams) (request.APIResponse[User], error) {
	params.JSON = getUserPayload{
		requestPayload: requestPayload{
			AuthKey: params.APIKey,
			Type:    "GetUser",
		},
	}

	response := &Response[User]{}
	res, err := c.Request("POST", "/api/getUser", params, response)
	return request.NewAPIResponse(res, response.Result), err
}

type getAddonsPayload struct {
	requestPayload
	Update bool `json:"update"`
}

type GetAddonsParams struct {
	Ctx
}

func (c Client) GetAddons(params *GetAddonsParams) (request.APIResponse[GetAddonsData], error) {
	params.JSON = getAddonsPayload{
		requestPayload: requestPayload{
			AuthKey: params.APIKey,
			Type:    "AddonCollectionGet",
		},
		Update: true,
	}

	response := &Response[GetAddonsData]{}
	res, err := c.Request("POST", "/api/addonCollectionGet", params, response)
	return request.NewAPIResponse(res, response.Result), err
}

type setAddonsPayload struct {
	requestPayload
	Addons []Addon `json:"addons"`
}

type SetAddonsParams struct {
	Ctx
	Addons []Addon
}

func (c Client) SetAddons(params *SetAddonsParams) (request.APIResponse[SetAddonsData], error) {
	params.JSON = setAddonsPayload{
		requestPayload: requestPayload{
			AuthKey: params.APIKey,
			Type:    "AddonCollectionSet",
		},
		Addons: params.Addons,
	}

	response := &Response[SetAddonsData]{}
	res, err := c.Request("POST", "/api/addonCollectionSet", params, response)
	return request.NewAPIResponse(res, response.Result), err
}
