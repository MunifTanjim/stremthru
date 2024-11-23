package alldebrid

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClient struct {
	Name             store.StoreName
	client           *APIClient
	listMagnetsCache core.Cache[string, []store.ListMagnetsDataItem]
}

func NewStore() *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{})
	c.Name = store.StoreNameAlldebrid

	c.listMagnetsCache = func() core.Cache[string, []store.ListMagnetsDataItem] {
		return core.NewCache[string, []store.ListMagnetsDataItem](&core.CacheConfig[string]{
			Name:     "store:alldebrid:listMagnets",
			HashKey:  core.CacheHashKeyString,
			Lifetime: 1 * time.Minute,
		})
	}()

	return c
}

func (c *StoreClient) getCacheKey(params store.RequestContext, key string) string {
	return params.GetAPIKey(c.client.apiKey) + ":" + key
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := c.client.GetUser(&GetUserParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:    res.Data.Username,
		Email: res.Data.Email,
	}
	if res.Data.IsPremium {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	} else if res.Data.IsTrial {
		data.SubscriptionStatus = store.UserSubscriptionStatusTrial
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, err
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	mi, err := c.client.GetMagnetInstant(&GetMagnetInstantParams{
		Ctx:     params.Ctx,
		Magnets: params.Magnets,
	})
	if err != nil {
		return nil, err
	}

	data := &store.CheckMagnetData{}

	for _, magnet := range mi.Data {
		item := &store.CheckMagnetDataItem{
			Magnet: magnet.Magnet,
			Hash:   magnet.Hash,
			Status: store.MagnetStatusUnknown,
		}

		if magnet.Error != nil {
			if magnet.Error.Code == MagnetErrorCodeInvalidURI {
				item.Status = store.MagnetStatusInvalid
			}
		} else if magnet.Instant {
			item.Status = store.MagnetStatusCached

			for _, file := range magnet.GetFiles() {
				if file.Type == store.MagnetFileTypeFolder || file.Size == 0 {
					continue
				}

				item.Files = append(item.Files, store.MagnetFile{
					Idx:  file.Idx,
					Name: file.Name,
					Size: file.Size,
				})
			}
		}

		data.Items = append(data.Items, *item)
	}

	return data, nil
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	um, err := c.client.UploadMagnet(&UploadMagnetParams{
		Ctx:     params.Ctx,
		Magnets: []string{params.Magnet},
	})
	if err != nil {
		return nil, err
	}

	c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

	magnet := um.Data[0]

	if magnet.Error != nil {
		return nil, UpstreamErrorWithCause(magnet.Error)
	}

	data := &store.AddMagnetData{
		Id:     strconv.Itoa(magnet.Id),
		Hash:   magnet.Hash,
		Magnet: magnet.Magnet,
		Name:   magnet.Name,
		Status: store.MagnetStatusQueued,
	}

	if magnet.Ready {
		data.Status = store.MagnetStatusDownloaded

		ms, err := c.client.GetMagnetStatus(&GetMagnetStatusParams{
			Ctx: params.Ctx,
			Id:  magnet.Id,
		})
		if err != nil {
			return nil, err
		}

		magnet := ms.Data

		for _, f := range magnet.GetFiles() {
			data.Files = append(data.Files, store.MagnetFile{
				Idx:  f.Idx,
				Link: f.Link,
				Name: f.Name,
				Path: f.Path,
				Size: f.Size,
			})
		}
	}

	return data, err
}

func statusCodeToMangetStatus(statusCode MagnetStatusCode) store.MagnetStatus {
	switch statusCode {
	case MagnetStatusCodeQueued:
		return store.MagnetStatusQueued
	case MagnetStatusCodeDownloading:
		return store.MagnetStatusDownloading
	case MagnetStatusCodeProcessing:
		return store.MagnetStatusProcessing
	case MagnetStatusCodeReady:
		return store.MagnetStatusDownloaded
	case MagnetStatusCodeUploading:
		return store.MagnetStatusUploading
	case MagnetStatusCodeErrorDeletedUpstream:
		fallthrough
	case MagnetStatusCodeErrorDownloadTimedOut:
		fallthrough
	case MagnetStatusCodeErrorDownloadTookTooLong:
		fallthrough
	case MagnetStatusCodeErrorFileTooBig:
		fallthrough
	case MagnetStatusCodeErrorUnknown:
		fallthrough
	case MagnetStatusCodeErrorUnpackFailed:
		return store.MagnetStatusFailed
	case MagnetStatusCodeUploadFailed:
		fallthrough
	default:
		return store.MagnetStatusUnknown
	}
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		error := core.NewStoreError("invalid id")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}

	ms, err := c.client.GetMagnetStatus(&GetMagnetStatusParams{
		Ctx: params.Ctx,
		Id:  id,
	})
	if err != nil {
		return nil, err
	}

	magnet := ms.Data

	data := &store.GetMagnetData{
		Id:     strconv.Itoa(magnet.Id),
		Hash:   magnet.Hash,
		Name:   magnet.Filename,
		Status: statusCodeToMangetStatus(magnet.StatusCode),
		Files:  []store.MagnetFile{},
	}

	for _, f := range magnet.GetFiles() {
		data.Files = append(data.Files, store.MagnetFile{
			Idx:  f.Idx,
			Link: f.Link,
			Name: f.Name,
			Path: f.Path,
			Size: f.Size,
		})
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	lm, found := c.listMagnetsCache.Get(c.getCacheKey(params, ""))
	if !found {
		res, err := c.client.GetAllMagnetStatus(&GetAllMagnetStatusParams{
			Ctx: params.Ctx,
		})
		if err != nil {
			return nil, err
		}

		items := []store.ListMagnetsDataItem{}
		for _, magnet := range res.Data.Magnets {
			item := &store.ListMagnetsDataItem{
				Id:     strconv.Itoa(magnet.Id),
				Hash:   magnet.Hash,
				Name:   magnet.Filename,
				Status: statusCodeToMangetStatus(magnet.StatusCode),
			}

			items = append(items, *item)
		}

		lm = items
		c.listMagnetsCache.Add(c.getCacheKey(params, ""), items)
	}

	totalItems := len(lm)
	startIdx := params.Offset
	if startIdx > totalItems {
		startIdx = totalItems
	}
	endIdx := startIdx + params.Limit
	if endIdx > totalItems {
		endIdx = totalItems
	}
	items := lm[startIdx:endIdx]

	data := &store.ListMagnetsData{
		Items:      items,
		TotalItems: totalItems,
	}

	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		error := core.NewStoreError("invalid id")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}

	_, err = c.client.DeleteMagnet(&DeleteMagnetParams{
		Ctx: params.Ctx,
		Id:  id,
	})
	if err != nil {
		return nil, err
	}

	c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

	data := &store.RemoveMagnetData{Id: params.Id}
	return data, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	ul, err := c.client.UnlockLink(&UnlockLinkParams{
		Ctx:  params.Ctx,
		Link: params.Link,
	})
	if err != nil {
		return nil, err
	}

	link := ul.Data

	if link.Delayed != 0 {
		error := core.NewStoreError("link generation delayed, try later")
		error.StatusCode = http.StatusTeapot
		return nil, error
	}

	data := &store.GenerateLinkData{
		Link: link.Link,
	}

	return data, nil
}
