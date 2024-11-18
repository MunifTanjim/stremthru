package realdebrid

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
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

type StoreClient struct {
	Name             store.StoreName
	client           *APIClient
	checkMagnetCache core.Cache[string, store.CheckMagnetDataItem]
	getMagnetCache   core.Cache[string, store.GetMagnetData] // for downloaded magnets
	idsByHashCache   core.Cache[string, map[string]bool]
	hashByIdCache    core.Cache[string, string]
}

func NewStoreClient() *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{})
	c.Name = store.StoreNameRealDebrid

	c.checkMagnetCache = func() core.Cache[string, store.CheckMagnetDataItem] {
		return core.NewCache[string, store.CheckMagnetDataItem](&core.CacheConfig[string]{
			Name:     "store:realdebrid:checkMagnet",
			HashKey:  core.CacheHashKeyString,
			Lifetime: 10 * time.Minute,
		})
	}()

	c.getMagnetCache = func() core.Cache[string, store.GetMagnetData] {
		return core.NewCache[string, store.GetMagnetData](&core.CacheConfig[string]{
			Name:     "store:realdebrid:getMagnet",
			HashKey:  core.CacheHashKeyString,
			Lifetime: 10 * time.Minute,
		})
	}()

	c.idsByHashCache = func() core.Cache[string, map[string]bool] {
		return core.NewCache[string, map[string]bool](&core.CacheConfig[string]{
			Name:     "store:realdebrid:idsByHash",
			HashKey:  core.CacheHashKeyString,
			Lifetime: 10 * time.Minute,
		})
	}()
	c.hashByIdCache = func() core.Cache[string, string] {
		return core.NewCache[string, string](&core.CacheConfig[string]{
			Name:     "store:realdebrid:hashById",
			HashKey:  core.CacheHashKeyString,
			Lifetime: 10 * time.Minute,
		})
	}()

	return c
}

func (c *StoreClient) getCacheKey(params store.RequestContext, key string) string {
	return params.GetAPIKey(c.client.apiKey) + ":" + key
}

func (c *StoreClient) addIdHashMapCache(params store.RequestContext, id, hash string) {
	c.hashByIdCache.Add(c.getCacheKey(params, id), hash)
	if ids, ok := c.idsByHashCache.Get(c.getCacheKey(params, hash)); ok {
		ids[id] = true
		c.idsByHashCache.Add(c.getCacheKey(params, hash), ids)
	} else {
		c.idsByHashCache.Add(c.getCacheKey(params, hash), map[string]bool{id: true})
	}
}

func (c *StoreClient) removeIdHashMapCache(params store.RequestContext, id, hash string) {
	c.hashByIdCache.Remove(c.getCacheKey(params, id))
	if ids, ok := c.idsByHashCache.Get(c.getCacheKey(params, hash)); ok {
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

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}

	tIdsMap, found := c.idsByHashCache.Get(c.getCacheKey(params, magnet.Hash))
	if !found {
		_, err := c.ListMagnets(&store.ListMagnetsParams{
			Ctx: params.Ctx,
		})
		if err != nil {
			return nil, err
		}
		tIdsMap, found = c.idsByHashCache.Get(c.getCacheKey(params, magnet.Hash))
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
		Id:     t.Id,
		Hash:   magnet.Hash,
		Magnet: magnet.Link,
		Name:   magnet.Name,
		Status: m.Status,
		Files:  m.Files,
	}
	return data, nil
}

func (c *StoreClient) getCachedCheckMagnet(params *store.CheckMagnetParams, magnetHash string) *store.CheckMagnetDataItem {
	if v, ok := c.checkMagnetCache.Get(params.GetAPIKey(c.client.apiKey) + ":" + magnetHash); ok {
		return &v
	}
	return nil
}

func (c *StoreClient) setCachedCheckMagnet(params *store.CheckMagnetParams, magnetHash string, v *store.CheckMagnetDataItem) {
	c.checkMagnetCache.Add(params.GetAPIKey(c.client.apiKey)+":"+magnetHash, *v)
}

func (c *StoreClient) checkMagnetInstantAvailability(params *store.CheckMagnetParams, hashes []string) (APIResponse[CheckTorrentInstantAvailabilityData], error) {
	res, err := c.client.CheckTorrentInstantAvailability(&CheckTorrentInstantAvailabilityParams{
		Ctx:    params.Ctx,
		Hashes: hashes,
	})
	return res, err
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	magnetByHash := map[string]core.MagnetLink{}
	hashes := []string{}

	cachedItemByHash := map[string]store.CheckMagnetDataItem{}
	uncachedHashes := []string{}

	for _, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		magnetByHash[magnet.Hash] = magnet
		hashes = append(hashes, magnet.Hash)
		if v := c.getCachedCheckMagnet(params, magnet.Hash); v != nil {
			cachedItemByHash[magnet.Hash] = *v
		} else {
			uncachedHashes = append(uncachedHashes, magnet.Hash)
		}
	}
	tByHash := map[string]CheckTorrentInstantAvailabilityDataHosterMap{}
	if len(uncachedHashes) > 0 {
		res, err := c.client.CheckTorrentInstantAvailability(&CheckTorrentInstantAvailabilityParams{
			Ctx:    params.Ctx,
			Hashes: uncachedHashes,
		})
		if err != nil {
			return nil, err
		}
		for hash, t := range res.Data {
			tByHash[strings.ToLower(hash)] = t
		}
	}

	data := &store.CheckMagnetData{}
	for _, hash := range hashes {
		if item, ok := cachedItemByHash[hash]; ok {
			data.Items = append(data.Items, item)
			continue
		}

		m := magnetByHash[hash]
		item := store.CheckMagnetDataItem{
			Hash:   m.Hash,
			Magnet: m.Link,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}
		if t, ok := tByHash[hash]; ok {
			largestVariant := map[string]CheckTorrentInstantAvailabilityDataFileIdsVariantFile{}
			largestVariantLength := 0

			for _, variants := range t {
				for _, fMap := range variants {
					length := len(fMap)
					if length > largestVariantLength {
						largestVariantLength = length
						largestVariant = fMap
					}
				}
			}

			for id, f := range largestVariant {
				idx, err := strconv.Atoi(id)
				if err != nil {
					return nil, err
				}
				item.Files = append(item.Files, store.MagnetFile{
					Idx:  idx - 1,
					Name: f.Filename,
					Size: f.Filesize,
				})
			}

			if largestVariantLength > 0 {
				item.Status = store.MagnetStatusCached
				c.setCachedCheckMagnet(params, hash, &item)
			}
		}

		data.Items = append(data.Items, item)
	}

	return data, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	res, err := c.client.UnrestrictLink(&UnrestrictLinkParams{
		Ctx:  params.Ctx,
		Link: params.Link,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GenerateLinkData{
		Link: res.Data.Download,
	}
	return data, nil
}

func (c *StoreClient) getCachedGetMagnet(params store.RequestContext, id string) *store.GetMagnetData {
	if v, ok := c.getMagnetCache.Get(params.GetAPIKey(c.client.apiKey) + ":" + id); ok {
		return &v
	}
	return nil
}

func (c *StoreClient) setCachedGetMagnet(params store.RequestContext, id string, v *store.GetMagnetData) {
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
		Id:     res.Data.Id,
		Hash:   res.Data.Hash,
		Name:   res.Data.Filename,
		Status: torrentStatusToMagnetStatus(res.Data.Status),
		Files:  []store.MagnetFile{},
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
				data.Files = append(data.Files, store.MagnetFile{
					Idx:  f.Id - 1,
					Name: filepath.Base(f.Path),
					Path: f.Path,
					Size: f.Bytes,
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
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.ListMagnetsData{
		Items: []store.ListMagnetsDataItem{},
	}
	for _, t := range res.Data {
		item := store.ListMagnetsDataItem{
			Id:     t.Id,
			Hash:   t.Hash,
			Name:   t.Filename,
			Status: torrentStatusToMagnetStatus(t.Status),
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
	if hash, ok := c.hashByIdCache.Get(c.getCacheKey(params, params.Id)); ok {
		c.removeIdHashMapCache(params, params.Id, hash)
	}
	return data, nil
}
