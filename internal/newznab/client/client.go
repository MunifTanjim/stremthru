package newznab_client

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type ClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	UserAgent  string
}

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client

	userAgent string
	apiKey    string

	reqQuery  func(query *url.Values, params request.Context)
	reqHeader func(query *http.Header, params request.Context)

	caps *cache.CachedValue[Caps]
}

func NewClient(conf *ClientConfig) *Client {
	if conf.HTTPClient == nil {
		conf.HTTPClient = config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	}

	c := Client{
		HTTPClient: conf.HTTPClient,
		userAgent:  conf.UserAgent,
		apiKey:     conf.APIKey,
	}

	c.BaseURL = util.MustParseURL(conf.BaseURL)

	c.reqQuery = func(query *url.Values, params request.Context) {
		query.Set("apikey", c.apiKey)
	}

	c.reqHeader = func(header *http.Header, params request.Context) {
		header.Set("User-Agent", c.userAgent)
	}

	c.caps = cache.NewCachedValue(cache.CachedValueConfig[Caps]{
		Get: func() (Caps, error) {
			res, err := c.getCaps(&GetCapsParams{})
			return res.Data, err
		},
		TTL: 15 * time.Minute,
	})

	return &c
}

type Response[T any] struct {
	Error *znab.Error
	Data  T
}

func (r Response[T]) GetError(res *http.Response) error {
	if r.Error == nil {
		return nil
	}
	return r.Error
}

func (r *Response[T]) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/xml") || strings.Contains(contentType, "application/rss+xml") || strings.Contains(contentType, "text/xml"):
		var root struct {
			XMLName xml.Name
		}
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		switch root.XMLName.Local {
		case "error":
			var xmlError znab.Error
			if err := xml.Unmarshal(body, &xmlError); err != nil {
				return err
			}
			r.Error = &xmlError
			return nil
		default:
			err := xml.Unmarshal(body, &r.Data)
			if err != nil {
				return err
			}
			return nil
		}
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

type Ctx = request.Ctx

func (c *Client) Request(method, path string, params request.Context, v request.ResponseContainer) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewAPIError("failed to create request")
		error.Cause = err
		return nil, error
	}
	res, err := params.DoRequest(c.HTTPClient, req)
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

func NewQuery(caps *Caps) *Query {
	return &Query{caps: caps, values: url.Values{}}
}

func (c *Client) NewSearchQuery(fn func(caps *Caps) Function) (*Query, error) {
	caps, err := c.GetCaps()
	if err != nil {
		return nil, err
	}
	q := NewQuery(&caps).SetT(fn(&caps))
	return q, nil
}
