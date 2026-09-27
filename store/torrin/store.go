package torrin

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

var (
	_ store.Store     = (*StoreClient)(nil)
	_ store.NewzStore = (*StoreClient)(nil)
)

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
	for itemIndex := range response.Data.Items {
		item := &response.Data.Items[itemIndex]
		for fileIndex := range item.Files {
			file := &item.Files[fileIndex]
			if isSignedLink(file.Link) {
				file.Link = generateFakeLink(item.Hash, file.Idx)
			}
		}
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
	data, err := s.getRawMagnet(params)
	if err != nil {
		return nil, err
	}
	// create placeholder link that can later be used to create presigned link
	// this is important for link stability, as Torrin returns signed link by
	// default
	for i := range data.Files {
		file := &data.Files[i]
		if isSignedLink(file.Link) {
			file.Link = generateFakeLink(data.Id, file.Idx)
		}
	}
	return data, nil
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
	if parsedId, success := parseIdFromFakeLink(params.Link); success {
		data, err := s.getRawMagnet(&store.GetMagnetParams{
			Ctx:      params.Ctx,
			Id:       parsedId.Id,
			ClientIP: params.ClientIP,
		})
		if err != nil {
			return nil, err
		}
		files := data.Files
		if len(files) <= parsedId.Idx {
			return nil, errors.New("No matching files found")
		}
		return &store.GenerateLinkData{
			Link: data.Files[parsedId.Idx].Link,
		}, nil
	}

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

func (s *StoreClient) CheckNewz(params *store.CheckNewzParams) (*store.CheckNewzData, error) {
	ctx := params.Ctx
	query := url.Values{"hash": params.Hashes}
	ctx.Query = &query
	response := &Response[store.CheckNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, "/v0/store/newz/check", &ctx, response)
	stats.Record(s.Name, "check_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	for itemIndex := range response.Data.Items {
		item := &response.Data.Items[itemIndex]
		for fileIndex := range item.Files {
			file := &item.Files[fileIndex]
			if isSignedLink(file.Link) {
				file.Link = generateFakeLink(item.Hash, file.Idx)
			}
		}
	}
	return &response.Data, nil
}

func (s *StoreClient) AddNewz(params *store.AddNewzParams) (*store.AddNewzData, error) {
	ctx := params.Ctx
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	if params.File != nil {
		form := multipart.Form{}
		form.File = make(map[string][]*multipart.FileHeader, 1)
		form.File["file"] = []*multipart.FileHeader{params.File}
		ctx.MultiPartForm = &form
	} else if params.Link != "" {
		ctx.JSON = map[string]string{"link": params.Link}
	} else {
		return nil, errors.New("File or link required")
	}
	response := &Response[store.AddNewzData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/newz", &ctx, response)
	stats.Record(s.Name, "add_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	data := &response.Data
	return data, nil
}

func (s *StoreClient) GetNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
	data, err := s.getRawNewz(params)
	if err != nil {
		return nil, err
	}
	// create placeholder link that can later be used to create presigned link
	// this is important for link stability, as Torrin returns signed link by
	// default
	for i := range data.Files {
		file := &data.Files[i]
		if isSignedLink(file.Link) {
			file.Link = generateFakeLink(data.Id, file.Idx)
		}
	}
	return data, nil
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
	if parsedId, success := parseIdFromFakeLink(params.Link); success {
		data, err := s.getRawNewz(&store.GetNewzParams{
			Ctx:      params.Ctx,
			Id:       parsedId.Id,
			ClientIP: params.ClientIP,
		})
		if err != nil {
			return nil, err
		}
		files := data.Files
		if len(files) <= parsedId.Idx {
			return nil, errors.New("No matching files found")
		}
		return &store.GenerateNewzLinkData{
			Link: data.Files[parsedId.Idx].Link,
		}, nil
	}

	ctx := params.Ctx
	if params.ClientIP != "" {
		ctx.Query = &url.Values{"client_ip": []string{params.ClientIP}}
	}
	ctx.JSON = map[string]string{"link": params.Link}
	response := &Response[store.GenerateNewzLinkData]{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, "/v0/store/newz/link/generate", &ctx, response)
	stats.Record(s.Name, "generate_newz_link", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

type parsedId struct {
	Id  string
	Idx int
}

func isSignedLink(link string) bool {
	return strings.Contains(link, "expires=")
}

func generateFakeLink(id string, idx int) string {
	return fmt.Sprintf("torrin://file-by-id/%s/%d", id, idx)
}

func parseIdFromFakeLink(link string) (*parsedId, bool) {
	idAndIdxStr, found := strings.CutPrefix(link, "torrin://file-by-id/")
	if !found {
		return nil, false
	}

	idAndIdx := strings.Split(idAndIdxStr, "/")
	if len(idAndIdx) != 2 {
		return nil, false
	}

	id := idAndIdx[0]
	idx, err := strconv.Atoi(idAndIdx[1])
	if err != nil {
		return nil, false
	}

	return &parsedId{Id: id, Idx: idx}, true
}

func (s *StoreClient) getRawMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
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

func (s *StoreClient) getRawNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
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
