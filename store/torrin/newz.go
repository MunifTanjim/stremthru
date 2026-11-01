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

var _ store.NewzStore = (*StoreClient)(nil)

func (s *StoreClient) CheckNewz(params *store.CheckNewzParams) (*store.CheckNewzData, error) {
	ctx := params.Ctx
	ctx.Query = &url.Values{"hash": params.Hashes}
	response := &Response[store.CheckNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/newz/check", &ctx, response)
	stats.Record(s.Name, "check_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) AddNewz(params *store.AddNewzParams) (*store.AddNewzData, error) {
	if params.Link == "" {
		err := core.NewStoreError("nzb file upload is not supported")
		err.StoreName = string(s.Name)
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}
	ctx := params.Ctx
	ctx.JSON = map[string]string{"link": params.Link}
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.AddNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/newz", &ctx, response)
	stats.Record(s.Name, "add_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) GetNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
	ctx := params.Ctx
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.GetNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/newz/"+url.PathEscape(params.Id), &ctx, response)
	stats.Record(s.Name, "get_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) ListNewz(params *store.ListNewzParams) (*store.ListNewzData, error) {
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
	response := &Response[store.ListNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/newz", &ctx, response)
	stats.Record(s.Name, "list_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (s *StoreClient) RemoveNewz(params *store.RemoveNewzParams) (*store.RemoveNewzData, error) {
	ctx := params.Ctx
	response := &Response[store.RemoveNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodDelete, "/v0/store/newz/"+url.PathEscape(params.Id), &ctx, response)
	stats.Record(s.Name, "remove_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if response.Data.Id == "" {
		response.Data.Id = params.Id
	}
	return &response.Data, nil
}

func (s *StoreClient) GenerateNewzLink(params *store.GenerateNewzLinkParams) (*store.GenerateNewzLinkData, error) {
	ctx := params.Ctx
	ctx.JSON = map[string]string{"link": params.Link}
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	response := &Response[store.GenerateNewzLinkData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/newz/link/generate", &ctx, response)
	stats.Record(s.Name, "generate_newz_link", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}
