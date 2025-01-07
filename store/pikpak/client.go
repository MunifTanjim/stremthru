package pikpak

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

var DefaultHTTPTransport = request.DefaultHTTPTransport
var DefaultHTTPClient = request.DefaultHTTPClient

type APIClientConfig struct {
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

type APIClient struct {
	HTTPClient *http.Client
	apiKey     string
	agent      string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.UserAgent == "" {
		conf.UserAgent = "stremthru"
	}

	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &APIClient{}

	c.HTTPClient = conf.HTTPClient
	c.apiKey = conf.APIKey
	c.agent = conf.UserAgent

	c.reqQuery = func(query *url.Values, params request.Context) {
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Add("X-Client-Id", clientId)
		header.Add("X-Client-Version", clientVersion)
	}

	return c
}

type Ctx struct {
	request.Ctx
	auth     *AuthState `json:"-"`
	deviceId string     `json:"-"`
}

type Context interface {
	PreparePikpakHeader(header *http.Header)
}

func (ctx Ctx) PreparePikpakHeader(header *http.Header) {
	if ctx.auth == nil {
		return
	}
	if ctx.auth.AccessToken != "" {
		header.Add("Authorization", "Bearer "+ctx.auth.AccessToken)
	}
	if ctx.auth.CaptchaToken != "" {
		header.Add("X-Captcha-Token", ctx.auth.CaptchaToken)
		header.Add("User-Agent", buildUserAgent(ctx.GetDeviceId(), ctx.auth.UserId))
	} else {
		header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	}
	header.Add("X-Device-Id", ctx.GetDeviceId())
	// header.Add("X-Device-Sign", "wdi10."+ctx.DeviceId+"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
}

type PikpakUser struct {
	Username string
	Password string
}

func (ctx Ctx) GetDeviceId() string {
	if ctx.deviceId == "" {
		ctx.deviceId = getDeviceId(ctx.APIKey)
	}
	return ctx.deviceId
}

func (ctx Ctx) GetUser() *PikpakUser {
	if username, password, ok := strings.Cut(ctx.APIKey, ":"); ok {
		return &PikpakUser{
			Username: username,
			Password: password,
		}
	}
	return &PikpakUser{}
}

func (u PikpakUser) GetDeviceId() string {
	if u.Username == "" || u.Password == "" {
		return ""
	}
	return getDeviceId(u.Username + ":" + u.Password)
}

func (c APIClient) doRequest(params request.Context, req *http.Request, v ResponseEnvelop) (*http.Response, error) {
	if ctx, ok := params.(Context); ok {
		ctx.PreparePikpakHeader(&req.Header)
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

func (c APIClient) UserRequest(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(UserAPIBaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNamePikPak)
		error.Cause = err
		return nil, error
	}
	return c.doRequest(params, req, v)
}

func (c APIClient) DriveRequest(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(DriveAPIBaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNamePikPak)
		error.Cause = err
		return nil, error
	}
	return c.doRequest(params, req, v)
}
