package offcloud

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClientConfig struct {
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name             store.StoreName
	client           *APIClient
	listMagnetsCache cache.Cache[[]store.ListMagnetsDataItem]
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameOffcloud

	c.listMagnetsCache = cache.NewCache[[]store.ListMagnetsDataItem](&cache.CacheConfig{
		Name:     "store:offcloud:listMagnets",
		Lifetime: 5 * time.Minute,
	})

	return c
}

func (c *StoreClient) getCacheKey(params request.Context, key string) string {
	return params.GetAPIKey(c.client.apiKey) + ":" + key
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	if params.Magnet == "" {
		return nil, errors.New("torrent file not supported")
	}

	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}

	cdl, err := s.findCloudDownloadByHash(params.Ctx, magnet.Hash, false, false)
	if err != nil {
		return nil, err
	}

	data := &store.AddMagnetData{
		Hash:   magnet.Hash,
		Magnet: magnet.Link,
		Size:   -1,
		Files:  []store.MagnetFile{},
	}

	if cdl != nil {
		data.Id = cdl.Id
		data.Name = cdl.Name
		data.Status = cdl.Status
		data.AddedAt = cdl.AddedAt
		data.Size = cdl.Size
	} else {
		res, err := s.client.AddCloudDownload(&AddCloudDownloadParams{
			Ctx: params.Ctx,
			URL: magnet.RawLink,
		})
		if err != nil {
			return nil, err
		}

		data.Id = res.Data.RequestId
		data.Name = res.Data.FileName
		data.Status = getMagnetStatus(res.Data.Status)
		data.AddedAt = time.Now()

		s.listMagnetsCache.Remove(s.getCacheKey(params, ""))
	}

	if data.Status == store.MagnetStatusDownloaded {
		res, err := s.client.ExploreCloudDownload(&ExploreCloudDownloadParams{
			Ctx:       params.Ctx,
			RequestId: data.Id,
		})
		if err != nil {
			return nil, err
		}
		for i := range res.Data.Files {
			f := &res.Data.Files[i]
			data.Files = append(data.Files, store.MagnetFile{
				Idx:  -1,
				Name: f.GetName(),
				Path: f.GetPath(),
				Size: f.Size,
				Link: f.URL,
			})
		}
	}

	return data, nil
}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	hashes := []string{}
	magnets := []string{}
	magnetByHash := map[string]core.MagnetLink{}
	for _, magnet := range params.Magnets {
		if m, err := core.ParseMagnetLink(magnet); err == nil {
			hashes = append(hashes, m.Hash)
			magnets = append(magnets, m.RawLink)
			magnetByHash[m.Hash] = m
		}
	}
	res, err := s.client.GetCacheInfo(&GetCacheInfoParams{
		Ctx:          params.Ctx,
		URLs:         magnets,
		IncludeFiles: true,
	})
	if err != nil {
		return nil, err
	}
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}
	source := string(s.GetName().Code())
	for i, hash := range hashes {
		r := &res.Data[i]
		m := magnetByHash[hash]
		item := store.CheckMagnetDataItem{
			Hash:   m.Hash,
			Magnet: m.Link,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}
		if r.Cached {
			item.Status = store.MagnetStatusCached
		}
		for i := range r.Files {
			f := r.Files[i]
			item.Files = append(item.Files, store.MagnetFile{
				Idx:    -1,
				Name:   f.Filename,
				Path:   "/" + filepath.Join(filepath.Join(f.Folder...), f.Filename),
				Size:   -1,
				Source: source,
			})
		}
		data.Items = append(data.Items, item)
	}
	return data, nil
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	data := &store.GenerateLinkData{Link: params.Link}
	return data, nil
}

func getMagnetStatus(status CloudDownloadStatus) store.MagnetStatus {
	switch status {
	case CloudDownloadStatusCreated:
		return store.MagnetStatusQueued
	case CloudDownloadStatusDownloading:
		return store.MagnetStatusDownloading
	case CloudDownloadStatusDownloaded:
		return store.MagnetStatusDownloaded
	case CloudDownloadStatusError:
		return store.MagnetStatusFailed
	default:
		return store.MagnetStatusUnknown
	}
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	res, err := s.client.ExploreCloudDownload(&ExploreCloudDownloadParams{
		Ctx:       params.Ctx,
		RequestId: params.Id,
	})
	if err != nil {
		return nil, err
	}
	item, err := s.findCloudDownloadByID(params.Ctx, params.Id, false)
	if err != nil {
		return nil, err
	}
	if item == nil {
		uerr := UpstreamErrorWithCause(errors.New("cloud download not found"))
		uerr.StatusCode = 404
		return nil, uerr
	}
	data := &store.GetMagnetData{
		Id:      params.Id,
		Name:    item.Name,
		Hash:    item.Hash,
		Size:    -1,
		Status:  item.Status,
		Files:   []store.MagnetFile{},
		AddedAt: item.AddedAt,
	}
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		data.Files = append(data.Files, store.MagnetFile{
			Idx:  -1,
			Name: f.GetName(),
			Path: f.GetPath(),
			Size: f.Size,
			Link: f.URL,
		})
	}
	return data, nil
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := s.client.GetAccountInfo(&GetAccountInfoParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:                 res.Data.UserID,
		SubscriptionStatus: store.UserSubscriptionStatusTrial,
	}
	if res.Data.IsPremium {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	}
	if exp, err := time.Parse(time.DateOnly, res.Data.ExpirationDate); err == nil && !exp.After(time.Now()) {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, nil
}

func (s *StoreClient) findCloudDownloadByID(ctx Ctx, id string, refresh bool) (*store.ListMagnetsDataItem, error) {
	items, err := s.getCloudHistory(ctx, refresh)
	if err != nil {
		return nil, err
	}
	for i := range items {
		item := &items[i]
		if strings.Contains(item.Id, id) {
			return item, nil
		}
	}
	if !refresh {
		return s.findCloudDownloadByID(ctx, id, true)
	}
	return nil, nil
}

func (s *StoreClient) findCloudDownloadByHash(ctx Ctx, hash string, mustFind, refresh bool) (*store.ListMagnetsDataItem, error) {
	items, err := s.getCloudHistory(ctx, refresh)
	if err != nil {
		return nil, err
	}
	for i := range items {
		item := &items[i]
		if strings.Contains(item.Hash, hash) {
			return item, nil
		}
	}
	if mustFind && !refresh {
		return s.findCloudDownloadByHash(ctx, hash, false, true)
	}
	return nil, nil
}

func (s *StoreClient) getCloudHistory(ctx Ctx, refresh bool) ([]store.ListMagnetsDataItem, error) {
	items := []store.ListMagnetsDataItem{}
	if refresh || !s.listMagnetsCache.Get(s.getCacheKey(&ctx, ""), &items) {
		res, err := s.client.GetCloudHistory(&GetCloudHistoryParams{
			Ctx: ctx,
		})
		if err != nil {
			return nil, err
		}
		for _, hitem := range res.Data {
			magnet, err := core.ParseMagnetLink(hitem.OriginalLink)
			if err != nil {
				continue
			}
			item := store.ListMagnetsDataItem{
				AddedAt: hitem.CreatedOn,
				Hash:    magnet.Hash,
				Id:      hitem.RequestId,
				Name:    hitem.FileName,
				Size:    -1,
				Status:  getMagnetStatus(hitem.Status),
			}
			items = append(items, item)
		}
		s.listMagnetsCache.Add(s.getCacheKey(&ctx, ""), items)
	}
	return items, nil
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	lm, err := s.getCloudHistory(params.Ctx, false)
	if err != nil {
		return nil, err
	}

	totalItems := len(lm)
	startIdx := min(params.Offset, totalItems)
	endIdx := min(startIdx+params.Limit, totalItems)
	items := lm[startIdx:endIdx]

	data := &store.ListMagnetsData{
		Items:      items,
		TotalItems: totalItems,
	}

	return data, nil
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	err := UpstreamErrorWithCause(errors.New("not supported"))
	err.StatusCode = http.StatusNotImplemented
	return nil, err
}
