package debridlink

import (
	"net/http"
	"net/url"
	"strings"
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
	Name                    store.StoreName
	client                  *APIClient
	subscriptionStatusCache cache.Cache[store.UserSubscriptionStatus]
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameDebridLink

	c.subscriptionStatusCache = cache.NewLRUCache[store.UserSubscriptionStatus](&cache.CacheConfig{
		Name:     "store:debridlink:subscriptionStatus",
		Lifetime: 5 * time.Minute,
	})

	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := c.client.GetAccountInfo(&GetAccountInfoParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	url, err := url.Parse(res.Data.ViewSessidUrl)
	if err != nil {
		return nil, err
	}
	id := strings.Split(url.Path, "/")[2]
	data := &store.User{
		Id:    id,
		Email: res.Data.Email,
	}
	if res.Data.PremiumLeft != 0 {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, err
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
	if status == store.UserSubscriptionStatusPremium {
		return nil
	}
	err := core.NewAPIError("forbidden")
	err.Code = core.ErrorCodeForbidden
	err.StatusCode = http.StatusForbidden
	return err
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	if !params.IsTrustedRequest {
		if err := c.assertValidSubscription(params.GetAPIKey(c.client.apiKey)); err != nil {
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

	data, err := buddy.CheckMagnet(c, hashes, params.GetAPIKey(c.client.apiKey), params.ClientIP, params.SId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}
	res, err := c.client.AddSeedboxTorrent(&AddSeedboxTorrentParams{
		Ctx:           params.Ctx,
		Url:           magnet.Link,
		StructureType: SeedboxTorrentStructureTypeList,
		IP:            params.ClientIP,
	})
	if err != nil {
		return nil, err
	}

	t := res.Data

	data := &store.AddMagnetData{
		Id:      t.Id,
		Hash:    t.HashString,
		Magnet:  magnet.Link,
		Name:    t.Name,
		Size:    t.TotalSize,
		Status:  store.MagnetStatusQueued,
		Files:   []store.MagnetFile{},
		AddedAt: t.GetAddedAt(),
	}

	if t.DownloadPercent == 100 {
		data.Status = store.MagnetStatusDownloaded

		for idx, f := range t.Files {
			file := &store.MagnetFile{
				Idx:  idx,
				Name: f.Name,
				Size: f.Size,
				Path: "/" + f.Name,
				Link: f.DownloadUrl,
			}

			data.Files = append(data.Files, *file)
		}
	}

	return data, nil
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	res, err := c.client.ListSeedboxTorrents(&ListSeedboxTorrentsParams{
		Ctx: params.Ctx,
		Ids: []string{params.Id},
		IP:  params.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data.Value) != 1 || res.Data.Value[0].Id != params.Id {
		err := core.NewAPIError("not found")
		err.StatusCode = http.StatusNotFound
		err.StoreName = string(store.StoreNameDebridLink)
		return nil, err
	}
	t := res.Data.Value[0]
	data := &store.GetMagnetData{
		Id:      t.Id,
		Hash:    t.HashString,
		Name:    t.Name,
		Size:    t.TotalSize,
		Status:  store.MagnetStatusUnknown,
		Files:   []store.MagnetFile{},
		AddedAt: t.GetAddedAt(),
	}
	if t.DownloadPercent == 100 {
		data.Status = store.MagnetStatusDownloaded
	} else if t.DownloadPercent < 100 {
		data.Status = store.MagnetStatusDownloading
	}
	for idx, f := range t.Files {
		file := &store.MagnetFile{
			Idx:  idx,
			Link: f.DownloadUrl,
			Name: f.Name,
			Path: "/" + f.Name,
			Size: f.Size,
		}

		data.Files = append(data.Files, *file)
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	origLimit := params.Limit
	origOffset := params.Offset

	data := &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: 0,
	}
	totalPages := 0

	limit := LIST_SEEDBOX_TORRENTS_PER_PAGE_MAX
	page := origOffset / limit
	offsetInPage := origOffset % limit
	remainingItems := origLimit
	for remainingItems > 0 {
		res, err := c.client.ListSeedboxTorrents(&ListSeedboxTorrentsParams{
			Ctx:     params.Ctx,
			PerPage: limit,
			Page:    page,
			IP:      params.ClientIP,
		})
		if err != nil {
			return nil, err
		}

		resItems := res.Data.Value
		totalPages = res.Data.Pagination.Pages
		totalResItems := len(resItems)
		if totalResItems == 0 {
			break
		}

		if offsetInPage != 0 {
			resItems = resItems[offsetInPage:]
			totalResItems = len(resItems)
			offsetInPage = 0
		}

		for _, t := range resItems[:min(totalResItems, remainingItems)] {
			item := &store.ListMagnetsDataItem{
				Id:      t.Id,
				Hash:    t.HashString,
				Name:    t.Name,
				Size:    t.TotalSize,
				Status:  store.MagnetStatusUnknown,
				AddedAt: t.GetAddedAt(),
			}
			if t.DownloadPercent == 100 {
				item.Status = store.MagnetStatusDownloaded
			} else if t.DownloadPercent < 100 {
				item.Status = store.MagnetStatusDownloading
			}

			data.Items = append(data.Items, *item)
		}

		page++
		remainingItems -= totalResItems
	}

	data.TotalItems = totalPages * limit

	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	res, err := c.client.RemoveSeedboxTorrents(&RemoveSeedboxTorrentParams{
		Ctx: params.Ctx,
		Ids: []string{params.Id},
	})
	if err != nil {
		return nil, err
	}
	data := &store.RemoveMagnetData{
		Id: res.Data[0],
	}
	return data, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	data := &store.GenerateLinkData{Link: params.Link}
	return data, nil
}
