package realdebrid

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

func torrentStatusToMagnetStatus(status TorrentStatus) store.MagnetStatus {
	switch status {
	case TorrentStatusMagnetError:
		return store.MagnetStatusFailed
	case TorrentStatusMagnetConversion:
		return store.MagnetStatusInvalid
	case TorrentStatusWaitingFilesSelection:
		return store.MagnetStatusQueued
	case TorrentStatusQueued:
		return store.MagnetStatusQueued
	case TorrentStatusDownloading:
		return store.MagnetStatusDownloading
	case TorrentStatusDownloaded:
		return store.MagnetStatusDownloaded
	case TorrentStatusError:
		return store.MagnetStatusFailed
	case TorrentStatusVirus:
		return store.MagnetStatusFailed
	case TorrentStatusCompressing:
		return store.MagnetStatusProcessing
	case TorrentStatusUploading:
		return store.MagnetStatusUploading
	case TorrentStatusDead:
		return store.MagnetStatusFailed
	default:
		return store.MagnetStatusUnknown
	}
}

type StoreClientConfig struct {
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name                    store.StoreName
	client                  *APIClient
	getMagnetCache          cache.Cache[store.GetMagnetData] // for downloaded magnets
	idsByHashCache          cache.Cache[map[string]bool]
	hashByIdCache           cache.Cache[string]
	subscriptionStatusCache cache.Cache[store.UserSubscriptionStatus]
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameRealDebrid

	c.getMagnetCache = func() cache.Cache[store.GetMagnetData] {
		return cache.NewCache[store.GetMagnetData](&cache.CacheConfig{
			Name:     "store:realdebrid:getMagnet",
			Lifetime: 10 * time.Minute,
		})
	}()

	c.idsByHashCache = func() cache.Cache[map[string]bool] {
		return cache.NewCache[map[string]bool](&cache.CacheConfig{
			Name:     "store:realdebrid:idsByHash",
			Lifetime: 10 * time.Minute,
		})
	}()
	c.hashByIdCache = func() cache.Cache[string] {
		return cache.NewCache[string](&cache.CacheConfig{
			Name:     "store:realdebrid:hashById",
			Lifetime: 10 * time.Minute,
		})
	}()
	c.subscriptionStatusCache = cache.NewLRUCache[store.UserSubscriptionStatus](&cache.CacheConfig{
		Name:     "store:realdebrid:subscriptionStatus",
		Lifetime: 5 * time.Minute,
	})

	return c
}

func (c *StoreClient) getCacheKey(params request.Context, key string) string {
	return params.GetAPIKey(c.client.apiKey) + ":" + key
}

func (c *StoreClient) addIdHashMapCache(params request.Context, id, hash string) {
	c.hashByIdCache.Add(c.getCacheKey(params, id), hash)
	ids := map[string]bool{}
	if c.idsByHashCache.Get(c.getCacheKey(params, hash), &ids) {
		ids[id] = true
		c.idsByHashCache.Add(c.getCacheKey(params, hash), ids)
	} else {
		c.idsByHashCache.Add(c.getCacheKey(params, hash), map[string]bool{id: true})
	}
}

func (c *StoreClient) removeIdHashMapCache(params request.Context, id, hash string) {
	c.hashByIdCache.Remove(c.getCacheKey(params, id))
	ids := map[string]bool{}
	if c.idsByHashCache.Get(c.getCacheKey(params, hash), &ids) {
		delete(ids, id)
		if len(ids) == 0 {
			c.idsByHashCache.Remove(c.getCacheKey(params, hash))
		} else {
			c.idsByHashCache.Add(c.getCacheKey(params, hash), ids)
		}
	}
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
		Id:    strconv.Itoa(res.Data.Id),
		Email: res.Data.Email,
	}
	if res.Data.Premium > 0 {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, nil
}

func shouldRemoveTorrent(t *GetTorrentInfoData) bool {
	status := t.Status
	return (status == TorrentStatusMagnetError || status == TorrentStatusError || status == TorrentStatusVirus || status == TorrentStatusDead) || ((status == TorrentStatusQueued || status == TorrentStatusDownloading || status == TorrentStatusDownloaded) && len(getSelectedFileIdsFromTorrent(t)) != len(getVideoFileIdsFromTorrent(t)))
}

func (c *StoreClient) waitForTorrentStatus(ctx store.Ctx, t *GetTorrentInfoData, status TorrentStatus, maxRetry int, retryInterval time.Duration) (*GetTorrentInfoData, error) {
	retry := 0
	for t.Status != status && retry < maxRetry {
		tInfo, err := c.client.GetTorrentInfo(&GetTorrentInfoParams{
			Ctx: ctx,
			Id:  t.Id,
		})
		if err != nil {
			return nil, err
		}
		t = &tInfo.Data
		time.Sleep(retryInterval)
		retry++
	}
	if t.Status != status {
		error := core.NewStoreError("torrent failed to reach status: " + string(status))
		error.StoreName = string(store.StoreNameRealDebrid)
		return nil, error
	}
	return t, nil
}

func getSelectedFileIdsFromTorrent(t *GetTorrentInfoData) []string {
	fileIds := []string{}
	for _, f := range t.Files {
		if f.Selected == 1 {
			fileIds = append(fileIds, strconv.Itoa(f.Id))
		}
	}
	return fileIds
}

func getVideoFileIdsFromTorrent(t *GetTorrentInfoData) []string {
	fileIds := []string{}
	for _, f := range t.Files {
		if core.HasVideoExtension(f.Path) {
			fileIds = append(fileIds, strconv.Itoa(f.Id))
		}
	}
	return fileIds
}

func (f *GetTorrentInfoDataFile) toStoreMagnetFile() store.MagnetFile {
	return store.MagnetFile{
		Idx:  f.Id - 1,
		Name: filepath.Base(f.Path),
		Size: f.Bytes,
	}
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}

	tIdsMap := map[string]bool{}
	if !c.idsByHashCache.Get(c.getCacheKey(params, magnet.Hash), &tIdsMap) {
		_, err := c.ListMagnets(&store.ListMagnetsParams{
			Ctx: params.Ctx,
		})
		if err != nil {
			return nil, err
		}
		c.idsByHashCache.Get(c.getCacheKey(params, magnet.Hash), &tIdsMap)
	}
	var t *GetTorrentInfoData
	for tId := range tIdsMap {
		tInfo, err := c.client.GetTorrentInfo(&GetTorrentInfoParams{
			Ctx: params.Ctx,
			Id:  tId,
		})
		if err != nil {
			return nil, err
		}
		t = &tInfo.Data
		if shouldRemoveTorrent(&tInfo.Data) {
			_, err := c.RemoveMagnet(&store.RemoveMagnetParams{
				Ctx: params.Ctx,
				Id:  t.Id,
			})
			if err != nil {
				return nil, err
			}
			t = nil
		}
	}

	if t == nil {
		res, err := c.client.AddMagnet(&AddMagnetParams{
			Ctx:    params.Ctx,
			Magnet: magnet.Link,
			IP:     params.ClientIP,
		})
		if err != nil {
			return nil, err
		}
		tInfo, err := c.client.GetTorrentInfo(&GetTorrentInfoParams{
			Ctx: params.Ctx,
			Id:  res.Data.Id,
		})
		if err != nil {
			return nil, err
		}
		t = &tInfo.Data
	}

	if t.Status != TorrentStatusQueued && t.Status != TorrentStatusDownloading && t.Status != TorrentStatusDownloaded {
		t, err = c.waitForTorrentStatus(params.Ctx, t, TorrentStatusWaitingFilesSelection, 5, 5*time.Second)
		if err != nil {
			return nil, err
		}
		_, err = c.client.StartTorrentDownload(&StartTorrentDownloadParams{
			Ctx:     params.Ctx,
			Id:      t.Id,
			FileIds: getVideoFileIdsFromTorrent(t),
			IP:      params.ClientIP,
		})
		if err != nil {
			return nil, err
		}
	}

	m, err := c.GetMagnet(&store.GetMagnetParams{
		Ctx: params.Ctx,
		Id:  t.Id,
	})
	data := &store.AddMagnetData{
		Id:      t.Id,
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    m.Name,
		Size:    t.OriginalBytes,
		Status:  m.Status,
		Files:   m.Files,
		AddedAt: m.AddedAt,
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

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	res, err := c.client.UnrestrictLink(&UnrestrictLinkParams{
		Ctx:  params.Ctx,
		Link: params.Link,
		IP:   params.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GenerateLinkData{
		Link: res.Data.Download,
	}
	return data, nil
}

func (c *StoreClient) getCachedGetMagnet(params request.Context, id string) *store.GetMagnetData {
	v := store.GetMagnetData{}
	if c.getMagnetCache.Get(params.GetAPIKey(c.client.apiKey)+":"+id, &v) {
		return &v
	}
	return nil
}

func (c *StoreClient) setCachedGetMagnet(params request.Context, id string, v *store.GetMagnetData) {
	if v == nil {
		c.getMagnetCache.Remove(params.GetAPIKey(c.client.apiKey) + ":" + id)
		return
	}
	c.getMagnetCache.Add(params.GetAPIKey(c.client.apiKey)+":"+id, *v)
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	if v := c.getCachedGetMagnet(params, params.Id); v != nil {
		return v, nil
	}
	res, err := c.client.GetTorrentInfo(&GetTorrentInfoParams{
		Ctx: params.Ctx,
		Id:  params.Id,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GetMagnetData{
		Id:      res.Data.Id,
		Hash:    res.Data.Hash,
		Name:    res.Data.Filename,
		Size:    res.Data.OriginalBytes,
		Status:  torrentStatusToMagnetStatus(res.Data.Status),
		Files:   []store.MagnetFile{},
		AddedAt: res.Data.GetAddedAt(),
	}
	totalLinks := len(res.Data.Links)
	if data.Status == store.MagnetStatusDownloaded {
		idx := -1
		for _, f := range res.Data.Files {
			if f.Selected == 1 {
				idx++
				link := ""
				if totalLinks >= idx+1 {
					link = res.Data.Links[idx]
				}
				smFile := f.toStoreMagnetFile()
				data.Files = append(data.Files, store.MagnetFile{
					Idx:  smFile.Idx,
					Name: smFile.Name,
					Path: f.Path,
					Size: smFile.Size,
					Link: link,
				})
			}
		}
		c.setCachedGetMagnet(params, params.Id, data)
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	res, err := c.client.ListTorrents(&ListTorrentsParams{
		Ctx:    params.Ctx,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, err
	}

	sTotal := res.Header.Get("X-Total-Count")
	if sTotal == "" {
		sTotal = "0"
	}
	total, err := strconv.Atoi(sTotal)
	if err != nil {
		return nil, err
	}
	data := &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: total,
	}
	for _, t := range res.Data {
		item := store.ListMagnetsDataItem{
			Id:      t.Id,
			Hash:    t.Hash,
			Name:    t.Filename,
			Size:    t.Bytes,
			Status:  torrentStatusToMagnetStatus(t.Status),
			AddedAt: t.GetAddedAt(),
		}
		data.Items = append(data.Items, item)
		c.addIdHashMapCache(params, item.Id, item.Hash)
	}
	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	_, err := c.client.DeleteTorrent(&DeleteTorrentParams{
		Ctx: params.Ctx,
		Id:  params.Id,
	})
	if err != nil {
		return nil, err
	}
	data := &store.RemoveMagnetData{
		Id: params.Id,
	}
	c.setCachedGetMagnet(params, params.Id, nil)
	hash := ""
	if c.hashByIdCache.Get(c.getCacheKey(params, params.Id), &hash) {
		c.removeIdHashMapCache(params, params.Id, hash)
	}
	return data, nil
}
