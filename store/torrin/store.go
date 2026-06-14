package torrin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/store"
)

// Torrin implements the StremThru store API directly, so this client is a thin
// forwarder: each store method maps 1:1 to a /v0/store/* endpoint and decodes
// the standard {"data": ...} envelope into the shared store types.

const defaultBaseURL = "https://api.torrin.app"

var DefaultHTTPClient = config.DefaultHTTPClient

type StoreClientConfig struct {
	BaseURL    string // default: https://api.torrin.app
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name       store.StoreName
	baseURL    string
	httpClient *http.Client
	agent      string
}

var _ store.Store = (*StoreClient)(nil)

func NewStoreClient(conf *StoreClientConfig) *StoreClient {
	if conf == nil {
		conf = &StoreClientConfig{}
	}
	c := &StoreClient{Name: store.StoreNameTorrin}

	c.baseURL = strings.TrimRight(conf.BaseURL, "/")
	if c.baseURL == "" {
		c.baseURL = defaultBaseURL
	}

	c.httpClient = conf.HTTPClient
	if c.httpClient == nil {
		c.httpClient = DefaultHTTPClient
	}

	c.agent = conf.UserAgent
	if c.agent == "" {
		c.agent = "stremthru"
	}

	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) newError(msg string, statusCode int) error {
	err := core.NewStoreError(msg)
	err.StoreName = string(store.StoreNameTorrin)
	err.StatusCode = statusCode
	return err
}

// doRequest performs an authenticated call and decodes the {"data": T} envelope.
func doRequest[T any](c *StoreClient, ctx context.Context, method, path, apiKey string, query url.Values, body any) (*T, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, c.newError("failed to encode request", http.StatusInternalServerError)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
	if err != nil {
		return nil, c.newError("failed to create request", http.StatusInternalServerError)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", c.agent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.newError("request failed: "+err.Error(), http.StatusBadGateway)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, c.newError("failed to read response", http.StatusBadGateway)
	}

	if res.StatusCode >= 400 {
		var envErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &envErr)
		msg := envErr.Error.Message
		if msg == "" {
			msg = "upstream error"
		}
		return nil, c.newError(msg, res.StatusCode)
	}

	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, c.newError("failed to decode response", http.StatusBadGateway)
	}
	return &env.Data, nil
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	return doRequest[store.User](c, params.GetContext(), http.MethodGet, "/v0/store/user", params.GetAPIKey(""), nil, nil)
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	query := url.Values{}
	for _, m := range params.Magnets {
		query.Add("magnet", m)
	}
	if params.SId != "" {
		query.Set("sid", params.SId)
	}
	if params.ClientIP != "" {
		query.Set("client_ip", params.ClientIP)
	}
	return doRequest[store.CheckMagnetData](c, params.GetContext(), http.MethodGet, "/v0/store/magnets/check", params.GetAPIKey(""), query, nil)
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	body := map[string]string{"magnet": params.Magnet}
	return doRequest[store.AddMagnetData](c, params.GetContext(), http.MethodPost, "/v0/store/magnets", params.GetAPIKey(""), nil, body)
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	return doRequest[store.GetMagnetData](c, params.GetContext(), http.MethodGet, "/v0/store/magnets/"+url.PathEscape(params.Id), params.GetAPIKey(""), nil, nil)
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	query := url.Values{}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	return doRequest[store.ListMagnetsData](c, params.GetContext(), http.MethodGet, "/v0/store/magnets", params.GetAPIKey(""), query, nil)
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	return doRequest[store.RemoveMagnetData](c, params.GetContext(), http.MethodDelete, "/v0/store/magnets/"+url.PathEscape(params.Id), params.GetAPIKey(""), nil, nil)
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	body := map[string]string{"link": params.Link}
	return doRequest[store.GenerateLinkData](c, params.GetContext(), http.MethodPost, "/v0/store/link/generate", params.GetAPIKey(""), nil, body)
}
