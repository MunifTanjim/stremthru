package request

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var DefaultHTTPTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	return transport
}()

var DefaultHTTPClient = func() *http.Client {
	return &http.Client{
		Transport: DefaultHTTPTransport,
	}
}()

type Context interface {
	GetAPIKey(fallbackAPIKey string) string
	GetContext() context.Context
	PrepareHeader(header *http.Header)
	PrepareBody(method string, query *url.Values) (body io.Reader, contentType string, err error)
	NewRequest(baseURL *url.URL, method, path string, header func(header *http.Header, params Context), query func(query *url.Values, params Context)) (req *http.Request, err error)
}

type Ctx struct {
	APIKey  string          `json:"-"`
	Context context.Context `json:"-"`
	Form    *url.Values     `json:"-"`
	JSON    any             `json:"-"`
	Headers *http.Header    `json:"-"`
}

func (ctx Ctx) GetAPIKey(fallbackAPIKey string) string {
	if len(ctx.APIKey) > 0 {
		return ctx.APIKey
	}
	return fallbackAPIKey
}

func (ctx Ctx) GetContext() context.Context {
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	return ctx.Context
}

func (ctx Ctx) PrepareBody(method string, query *url.Values) (body io.Reader, contentType string, err error) {
	if ctx.JSON != nil {
		jsonBytes, err := json.Marshal(ctx.JSON)
		if err != nil {
			return nil, "", err
		}
		body = bytes.NewBuffer(jsonBytes)
		contentType = "application/json"
	}
	if ctx.Form != nil {
		if method == http.MethodHead || method == http.MethodGet || ctx.JSON != nil {
			for key, values := range *ctx.Form {
				for _, value := range values {
					query.Add(key, value)
				}
			}
		} else {
			body = strings.NewReader(ctx.Form.Encode())
			contentType = "application/x-www-form-urlencoded"
		}
	}
	return body, contentType, nil
}

func (ctx Ctx) PrepareHeader(header *http.Header) {
	if ctx.Headers == nil {
		return
	}

	for key, values := range *ctx.Headers {
		for _, value := range values {
			header.Add(key, value)
		}
	}
}

func (ctx Ctx) NewRequest(baseURL *url.URL, method, path string, header func(header *http.Header, params Context), query func(query *url.Values, params Context)) (req *http.Request, err error) {
	url := baseURL.JoinPath(path)

	q := url.Query()
	query(&q, ctx)

	body, contentType, err := ctx.PrepareBody(method, &q)
	if err != nil {
		return nil, err
	}

	url.RawQuery = q.Encode()

	req, err = http.NewRequestWithContext(ctx.GetContext(), method, url.String(), body)
	if err != nil {
		return nil, err
	}

	header(&req.Header, ctx)
	ctx.PrepareHeader(&req.Header)

	if len(contentType) > 0 {
		req.Header.Add("Content-Type", contentType)
	}

	return req, nil
}
