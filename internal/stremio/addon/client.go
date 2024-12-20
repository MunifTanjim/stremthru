package stremio_addon

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
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
	HTTPClient *http.Client
}

type Client struct {
	HTTPClient *http.Client

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewClient(conf *ClientConfig) *Client {
	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &Client{}

	c.HTTPClient = conf.HTTPClient

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
	}

	return c
}

type Ctx = request.Ctx

type ResponseError struct {
	Body       string `json:"body"`
	StatusCode int    `json:"status_code"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func processResponseBody(res *http.Response, err error, v any) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return &ResponseError{
			Body:       string(body),
			StatusCode: res.StatusCode,
		}
	}

	return core.UnmarshalJSON(res.StatusCode, body, v)
}

func (c Client) Request(method string, url *url.URL, params request.Context, v any) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(url, method, "", c.reqHeader, c.reqQuery)
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

type GetManifestParams struct {
	request.Ctx
	BaseURL *url.URL
}

func (c Client) GetManifest(params *GetManifestParams) (request.APIResponse[stremio.Manifest], error) {
	response := &stremio.Manifest{}
	res, err := c.Request("GET", params.BaseURL.JoinPath("manifest.json"), params, response)
	return request.NewAPIResponse(res, *response), err
}

type FetchStreamParams struct {
	request.Ctx
	BaseURL *url.URL
	Type    string
	Id      string
	Extra   string
}

func (c Client) FetchStream(params *FetchStreamParams) (request.APIResponse[stremio.StreamHandlerResponse], error) {
	response := &stremio.StreamHandlerResponse{}
	path := "stream/" + params.Type + "/" + params.Id
	if params.Extra != "" {
		path = path + "/" + params.Extra
	}
	res, err := c.Request("GET", params.BaseURL.JoinPath(path), params, response)
	return request.NewAPIResponse(res, *response), err
}

type ProxyResourceParams struct {
	request.Ctx
	BaseURL  *url.URL
	Resource string
	Type     string
	Id       string
	Extra    string
}

func (c Client) ProxyResource(w http.ResponseWriter, r *http.Request, params *ProxyResourceParams) {
	path := params.Resource + "/" + params.Type + "/" + params.Id
	if params.Extra != "" {
		path = path + "/" + params.Extra
	}
	shared.ProxyResponse(w, r, params.BaseURL.JoinPath(path).String())
}
