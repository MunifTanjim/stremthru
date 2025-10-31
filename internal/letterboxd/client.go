package letterboxd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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

	reqQuery   func(query *url.Values, params request.Context)
	reqHeader  func(query *http.Header, params request.Context)
	retryAfter time.Duration
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
		transport := config.DefaultHTTPTransport.Clone()
		transport.DisableKeepAlives = false
		conf.HTTPClient.Transport = transport
	}

	c := &APIClient{}

	c.BaseURL = util.MustParseURL("https://api.letterboxd.com/api")

	c.OAuth.Config = oauth.LetterboxdOAuthConfig.Config
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
		header.Set("Accept", "application/json")
		header.Set("Accept-Charset", "UTF-8")
		header.Set("Accept-Language", "en-US")
		header.Set("User-Agent", config.Integration.Letterboxd.UserAgent)
	}

	return c
}

type Ctx = request.Ctx

type ResponseError struct {
	Err     bool   `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (r *ResponseError) GetError(res *http.Response) error {
	if r == nil || !r.Err {
		return nil
	}
	return r
}

func (r *ResponseError) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	case strings.Contains(contentType, "text/plain") && res.StatusCode >= 400:
		r.Err = true
		r.Message = string(body)
		if code, ok := strings.CutPrefix(r.Message, "error code: "); ok {
			r.Code = code
		}
		return r
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

func (c *APIClient) GetRetryAfter() time.Duration {
	return c.retryAfter
}

var requestMutex sync.Mutex

func (c *APIClient) Request(method, path string, params request.Context, v request.ResponseContainer) (*http.Response, error) {
	requestMutex.Lock()
	defer requestMutex.Unlock()

	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	c.retryAfter = 0
	res, err := params.DoRequest(c.httpClient, req)
	err = request.ProcessResponseBody(res, err, v)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusTooManyRequests {
			retryAfter := res.Header.Get("Retry-After")
			c.retryAfter = time.Duration(util.SafeParseInt(retryAfter, 30)) * time.Second
		}
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
