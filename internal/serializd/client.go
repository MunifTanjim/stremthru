package serializd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type APIClientConfig struct {
	HTTPClient *http.Client
	Token      string
}

type APIClient struct {
	BaseURL    *url.URL
	httpClient *http.Client
	token      string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(header *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.DefaultHTTPClient
	}

	c := &APIClient{}

	c.BaseURL = util.MustParseURL("https://www.serializd.com")
	c.httpClient = conf.HTTPClient
	c.token = conf.Token

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Set("Accept", "application/json, text/plain, */*")
		header.Set("Origin", "https://www.serializd.com")
		header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
		header.Set("X-Requested-With", "serializd_vercel")
		if c.token != "" {
			header.Set("Cookie", "tvproject_credentials="+c.token)
		}
	}

	return c
}

type Ctx = request.Ctx

type ResponseError struct {
	Err string `json:"error,omitempty"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (r *ResponseError) GetError(res *http.Response) error {
	if res.StatusCode < 400 && (r == nil || r.Err == "") {
		return nil
	}
	if res.StatusCode >= 400 {
		if r == nil {
			r = &ResponseError{}
		}
		if r.Err == "" {
			r.Err = res.Status
		}
	}
	return r
}

func (r *ResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	case (strings.Contains(contentType, "text/html") || strings.Contains(contentType, "text/plain")) && res.StatusCode >= 400:
		r.Err = res.Status
		return nil
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

func (c APIClient) Request(method, path string, params request.Context, v request.ResponseContainer) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := params.DoRequest(c.httpClient, req)
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
