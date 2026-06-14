package torrin

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/stats"
)

// Torrin implements the StremThru store API directly, so this client is a thin
// forwarder: each store method maps 1:1 to a /v0/store/* endpoint and decodes
// the standard {"data": ...} envelope into the shared store types.

type StoreClientConfig struct {
	BaseURL    string // default: https://api.torrin.app
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
}

var _ store.Store = (*StoreClient)(nil)

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	if config == nil {
		config = &StoreClientConfig{}
	}
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		BaseURL:    config.BaseURL,
		APIKey:     config.APIKey,
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameTorrin

	return c
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	ctx := params.Ctx
	response := &Response[store.User]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/user", &ctx, response)
	stats.Record(s.Name, "get_user", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	ctx := params.Ctx
	query := url.Values{"magnet": params.Magnets}
	if params.SId != "" {
		query.Set("sid", params.SId)
	}
	if params.ClientIP != "" {
		query.Set("client_ip", params.ClientIP)
	}
	ctx.Query = &query
	response := &Response[store.CheckMagnetData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/magnets/check", &ctx, response)
	stats.Record(s.Name, "check_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	if params.Torrent != nil {
		err := core.NewStoreError("torrent file is not supported")
		err.StoreName = string(s.Name)
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}
	ctx := params.Ctx
	ctx.JSON = map[string]string{"magnet": params.Magnet}
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.AddMagnetData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/magnets", &ctx, response)
	stats.Record(s.Name, "add_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	data := &response.Data
	if data.Magnet == "" {
		if m, err := core.ParseMagnetLink(params.Magnet); err == nil {
			data.Magnet = m.RawLink
		}
	}
	return data, nil
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	ctx := params.Ctx
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.GetMagnetData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/magnets/"+url.PathEscape(params.Id), &ctx, response)
	stats.Record(s.Name, "get_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	ctx := params.Ctx
	query := url.Values{}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.ClientIP != "" {
		query.Set("client_ip", params.ClientIP)
	}
	ctx.Query = &query
	response := &Response[store.ListMagnetsData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/magnets", &ctx, response)
	stats.Record(s.Name, "list_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	ctx := params.Ctx
	response := &Response[store.RemoveMagnetData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodDelete, "/v0/store/magnets/"+url.PathEscape(params.Id), &ctx, response)
	stats.Record(s.Name, "remove_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if response.Data.Id == "" {
		response.Data.Id = params.Id
	}
	return &response.Data, nil
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	ctx := params.Ctx
	ctx.JSON = map[string]string{"link": params.Link}
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.GenerateLinkData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/link/generate", &ctx, response)
	stats.Record(s.Name, "generate_torz_link", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}
