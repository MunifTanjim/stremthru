package seedr

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClientConfig struct {
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient

	subscriptionStatusCache cache.Cache[store.UserSubscriptionStatus]
}

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	m, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}
	res, err := s.client.AddTask(&AddTaskParams{
		Ctx:           params.Ctx,
		TorrentMagnet: m.RawLink,
	})
	if err != nil {
		return nil, err
	}
	t, err := s.GetMagnet(&store.GetMagnetParams{
		Ctx: params.Ctx,
		Id:  res.Data.UserTorrentId,
	})
	if err != nil {
		return nil, err
	}
	data := &store.AddMagnetData{
		Id:      res.Data.UserTorrentId,
		Hash:    m.Hash,
		Magnet:  m.Link,
		Name:    t.Name,
		Size:    t.Size,
		Status:  t.Status,
		Files:   t.Files,
		Private: t.Private,
		AddedAt: t.AddedAt,
	}
	return data, nil
}

func (c *StoreClient) assertValidSubscription(apiKey string) error {
	var status store.UserSubscriptionStatus
	if !c.subscriptionStatusCache.Get(apiKey, &status) {
		params := &store.GetUserParams{}
		params.APIKey = apiKey
		user, err := c.GetUser(params)
		if err != nil {
			return err
		}
		status = user.SubscriptionStatus
		if err := c.subscriptionStatusCache.Add(apiKey, status); err != nil {
			return err
		}
	}
	if status != store.UserSubscriptionStatusExpired {
		return nil
	}
	err := core.NewAPIError("forbidden")
	err.Code = core.ErrorCodeForbidden
	err.StatusCode = http.StatusForbidden
	err.StatusCode = http.StatusForbidden
	return err
}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	if !params.IsTrustedRequest {
		if err := s.assertValidSubscription(params.GetAPIKey(s.client.apiKey)); err != nil {
			return nil, err
		}
	}

	hashes := []string{}
	for _, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, magnet.Hash)
	}

	data, err := buddy.CheckMagnet(s, hashes, params.GetAPIKey(s.client.apiKey), params.ClientIP, params.SId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GenerateLink implements store.Store.
func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	panic("unimplemented")
}

func getMagnetStatusFromTask(t *GetTaskContentsData) store.MagnetStatus {
	if t.Progress == 0 && t.Size == 0 && len(t.Files) == 0 {
		return store.MagnetStatusQueued
	}
	if 0 < t.Progress && t.Progress < 100 {
		return store.MagnetStatusDownloading
	}
	if t.Progress >= 100 {
		return store.MagnetStatusDownloaded
	}
	return store.MagnetStatusUnknown
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	task_id, err := strconv.ParseInt(params.Id, 10, 64)
	if err != nil {
		return nil, err
	}
	res, err := s.client.GetTaskContents(&GetTaskContentsParams{
		Ctx:    params.Ctx,
		TaskId: task_id,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GetMagnetData{
		Id:      strconv.FormatInt(res.Data.Id, 10),
		Name:    res.Data.GetName(),
		Hash:    res.Data.Hash,
		Size:    res.Data.Size,
		Status:  getMagnetStatusFromTask(&res.Data),
		Files:   []store.MagnetFile{},
		AddedAt: res.Data.GetCreatedAt(),
	}
	if data.Status == store.MagnetStatusDownloaded {
		source := string(s.GetName().Code())
		for i := range res.Data.Files {
			f := &res.Data.Files[i]
			file := store.MagnetFile{
				Idx:    -1,
				Path:   f.GetPath(),
				Name:   f.GetName(),
				Size:   f.Size,
				Source: source,
			}
			data.Files = append(data.Files, file)
		}
	} else {
		if data.Size == 0 {
			data.Size = -1
		}
	}
	return data, nil
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := s.client.GetUser(&GetUserParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:    strconv.Itoa(res.Data.Profile.Id),
		Email: res.Data.Profile.Email,
	}
	if res.Data.Account.IsPremium {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	} else if res.Data.Account.Storage.Used < res.Data.Account.Storage.Limit {
		data.SubscriptionStatus = store.UserSubscriptionStatusTrial
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, nil
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	res, err := s.client.GetTasks(&GetTasksParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	items := []store.ListMagnetsDataItem{}
	for i := range res.Data.Torrents {
		t := &res.Data.Torrents[i]
		item := store.ListMagnetsDataItem{
			Id:      strconv.FormatInt(t.Id, 10),
			Hash:    t.Hash,
			Name:    t.GetName(),
			Size:    t.Size,
			Status:  store.MagnetStatusUnknown,
			Private: t.IsPrivate,
			AddedAt: t.GetCreatedAt(),
		}
		if t.Progress == 0 && t.Size == 0 {
			item.Status = store.MagnetStatusQueued
		} else if 0 < t.Progress && t.Progress < 100 {
			item.Status = store.MagnetStatusDownloading
		} else if t.Progress >= 100 {
			item.Status = store.MagnetStatusDownloaded
		}
		items = append(items, item)
	}
	data := &store.ListMagnetsData{
		Items:      items,
		TotalItems: len(items),
	}
	return data, nil
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	task_id, err := strconv.ParseInt(params.Id, 10, 64)
	if err != nil {
		return nil, err
	}
	t_res, err := s.client.GetTask(&GetTaskParams{
		Ctx:    params.Ctx,
		TaskId: task_id,
	})
	if err != nil {
		return nil, err
	}
	_, err = s.client.DeleteTask(&DeleteTaskParams{
		Ctx:    params.Ctx,
		TaskId: t_res.Data.Id,
	})
	if err != nil {
		return nil, err
	}
	_, err = s.client.BatchDelete(&BatchDeleteParams{
		Ctx: params.Ctx,
		Items: []BatchDeleteParamsItem{
			{
				Id:   t_res.Data.FolderCreatedId,
				Type: "folder",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	data := &store.RemoveMagnetData{
		Id: params.Id,
	}
	return data, nil
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameSeedr

	c.subscriptionStatusCache = cache.NewLRUCache[store.UserSubscriptionStatus](&cache.CacheConfig{
		Name:     "store:seedr:subscriptionStatus",
		Lifetime: 5 * time.Minute,
	})

	return c
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}
