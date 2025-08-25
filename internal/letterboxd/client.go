package letterboxd

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/google/uuid"
)

type APIClientConfig struct {
	HTTPClient *http.Client
	apiKey     string
	secret     string
}

type APIClient struct {
	BaseURL    *url.URL
	httpClient *http.Client
	apiKey     string
	secret     string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.DefaultHTTPClient
	}

	c := &APIClient{}

	c.BaseURL = util.MustParseURL("https://api.letterboxd.com/api")

	c.httpClient = conf.HTTPClient
	c.apiKey = conf.apiKey
	c.secret = conf.secret

	c.reqQuery = func(query *url.Values, params request.Context) {
		query.Set("apikey", conf.apiKey)
		query.Set("nonce", uuid.NewString())
		query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
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
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

func (c APIClient) getRequestSignature(req *http.Request) (string, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return "", err
		}
	}

	req.Body = io.NopCloser(bytes.NewReader(body))

	data := []byte{}
	data = append(data, []byte(req.Method)...)
	data = append(data, 0)
	data = append(data, []byte(req.URL.String())...)
	data = append(data, 0)
	data = append(data, body...)

	mac := hmac.New(sha256.New, []byte(c.secret))
	_, err := mac.Write(data)
	if err != nil {
		return "", err
	}
	result := mac.Sum(nil)

	return hex.EncodeToString(result), nil
}

func (c APIClient) beforeRequest(req *http.Request) error {
	if req.URL.Host != c.BaseURL.Host {
		return nil
	}

	signature, err := c.getRequestSignature(req)
	if err != nil {
		return err
	}

	query := req.URL.Query()
	query.Set("signature", signature)
	req.URL.RawQuery = query.Encode()

	return nil
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
	params.BeforeDo(c.beforeRequest)
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

func Test() {
	res, err := client.Request("GET", "/v0/films/genres", nil, &ResponseError{})
	log.Debug("X", "res", res, "err", err)
}
