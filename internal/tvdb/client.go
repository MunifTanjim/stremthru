package tvdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
	"golang.org/x/oauth2"
)

type APIClientConfigOAuth struct {
	GetTokenSource func(oauth2.Config) oauth2.TokenSource
}

type APIClientConfig struct {
	HTTPClient *http.Client
	OAuth      *APIClientConfigOAuth
}

type APIClientOAuth struct {
	Config      oauth2.Config
	TokenSource oauth2.TokenSource
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

	c.BaseURL = util.MustParseURL("https://api4.thetvdb.com/v4")

	c.OAuth.Config = oauth.TVDBOAuthConfig.Config
	if conf.OAuth != nil {
		c.OAuth.TokenSource = conf.OAuth.GetTokenSource(c.OAuth.Config)
	}

	if c.OAuth.TokenSource == nil {
		c.httpClient = conf.HTTPClient
	} else {
		c.httpClient = oauth2.NewClient(
			context.WithValue(context.Background(), oauth2.HTTPClient, conf.HTTPClient),
			c.OAuth.TokenSource,
		)
	}

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
	}

	return c
}

type Ctx = request.Ctx

type ResponseError struct {
	Message string `json:"message"`
	Status  string `json:"status,omitempty"` // success / failure
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (r *ResponseError) GetError(res *http.Response) error {
	if r == nil || r.Status == "success" {
		return nil
	}
	return r
}

func (r *ResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
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
	res, err := c.httpClient.Do(req)
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

type Response[T any] struct {
	ResponseError
	Data T `json:"data"`
}
