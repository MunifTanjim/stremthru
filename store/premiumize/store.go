package premiumize

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClientConfig struct {
	ParentFolderName string
}

type StoreClient struct {
	Name             store.StoreName
	client           *APIClient
	config           *StoreClientConfig
	parentFolderId   string
	listMagnetsCache cache.Cache[[]store.ListMagnetsDataItem]
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	if config.ParentFolderName == "" {
		config.ParentFolderName = "stremthru"
	}

	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{})
	c.Name = store.StoreNamePremiumize
	c.config = config

	c.listMagnetsCache = func() cache.Cache[[]store.ListMagnetsDataItem] {
		return cache.NewCache[[]store.ListMagnetsDataItem](&cache.CacheConfig{
			Name:     "store:premiumize:listMagnets",
			Lifetime: 1 * time.Minute,
		})
	}()

	return c
}

func (c *StoreClient) getCacheKey(params request.Context, key string) string {
	return params.GetAPIKey(c.client.apiKey) + ":" + key
}

func (c *StoreClient) getFolderByName(apiKey string, folderName string) (*CreateFolderData, error) {
	if c.parentFolderId == "" && folderName != c.config.ParentFolderName {
		folder, err := c.getFolderByName(apiKey, c.config.ParentFolderName)
		if err != nil {
			return nil, err
		}
		if folder != nil {
			c.parentFolderId = folder.Id
		} else {
			params := &CreateFolderParams{Name: c.config.ParentFolderName}
			params.APIKey = apiKey
			res, err := c.client.CreateFolder(params)
			if err != nil {
				return nil, err
			}
			c.parentFolderId = res.Data.Id
		}
	}

	params := &ListFoldersParams{}
	params.APIKey = apiKey
	if c.parentFolderId != "" {
		params.Id = c.parentFolderId
	}
	res, err := c.client.ListFolders(params)
	if err != nil {
		return nil, err
	}

	for _, folder := range res.Data.Content {
		if folder.Name == folderName {
			return &CreateFolderData{Id: folder.Id}, nil
		}
	}

	return nil, nil
}

func (c *StoreClient) ensureFolder(apiKey string, name string) (*CreateFolderData, error) {
	folder, err := c.getFolderByName(apiKey, name)
	if err != nil {
		return nil, err
	}
	if folder != nil {
		return &CreateFolderData{Id: folder.Id}, nil
	}

	cf_params := &CreateFolderParams{Name: name}
	cf_params.ParentId = c.parentFolderId
	cf_params.APIKey = apiKey
	cf_res, err := c.client.CreateFolder(cf_params)
	if err != nil {
		return nil, err
	}
	return &cf_res.Data, nil
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
	data := &store.User{
		Id:    res.Data.CustomerId,
		Email: "",
	}
	if res.Data.PremiumUntil != 0 {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
		if res.Data.PremiumUntil < int(time.Now().Unix()) {
			data.SubscriptionStatus = store.UserSubscriptionStatusExpired
		}
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, err
}

func (c *StoreClient) getCachedMagnetFiles(apiKey string, magnet string, includeLink bool) ([]store.MagnetFile, error) {
	params := &CreateDirectDownloadLinkParams{Src: magnet}
	params.APIKey = apiKey
	res, err := c.client.CreateDirectDownloadLink(params)
	if err != nil {
		return nil, err
	}
	files := []store.MagnetFile{}
	for idx, f := range res.Data.Content {
		file := &store.MagnetFile{
			Idx:  idx,
			Name: filepath.Base(f.Path),
			Path: "/" + f.Path,
			Size: f.Size,
		}
		if includeLink {
			file.Link = f.Link
			if f.StreamLink != "" {
				file.Link = f.StreamLink
			}
		}
		files = append(files, *file)
	}
	return files, nil
}

func (c *StoreClient) checkMagnet(params *store.CheckMagnetParams, includeLink bool) (*store.CheckMagnetData, error) {
	res, err := c.client.CheckCache(&CheckCacheParams{
		Ctx:   params.Ctx,
		Items: params.Magnets,
	})
	if err != nil {
		return nil, err
	}

	data := &store.CheckMagnetData{}

	for idx, is_cached := range res.Data.Response {
		magnet, err := core.ParseMagnetLink(params.Magnets[idx])
		if err != nil {
			return nil, err
		}
		item := &store.CheckMagnetDataItem{
			Magnet: magnet.Link,
			Hash:   magnet.Hash,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}

		if is_cached {
			item.Status = store.MagnetStatusCached

			files, err := c.getCachedMagnetFiles(params.APIKey, item.Magnet, includeLink)
			if err != nil {
				item.Status = store.MagnetStatusUnknown
				continue
			}
			item.Files = files
		}

		data.Items = append(data.Items, *item)
	}

	return data, nil
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	return c.checkMagnet(params, false)
}

func getTransferById(c *StoreClient, apiKey string, id string) (*ListTransfersDataItem, error) {
	params := &ListTransfersParams{}
	params.APIKey = apiKey
	res, err := c.client.ListTransfers(params)
	if err != nil {
		return nil, err
	}
	for _, transfer := range res.Data.Transfers {
		if transfer.Id == id {
			return &transfer, nil
		}
	}
	return nil, nil
}

type CachedMagnetId string

const CachedMagnetIdPrefix = "premiumize:cached:magnet:"

func (id CachedMagnetId) isValid() bool {
	return strings.HasPrefix(string(id), CachedMagnetIdPrefix)
}

func (id CachedMagnetId) toId(value string) CachedMagnetId {
	if id.isValid() {
		return id
	}
	return CachedMagnetId(CachedMagnetIdPrefix + value)
}

func (id CachedMagnetId) toString() string {
	return string(id)
}

func (id CachedMagnetId) toHash() string {
	return strings.TrimPrefix(string(id), CachedMagnetIdPrefix)
}

func listFolderFlat(c *StoreClient, apiKey string, folderId string, result []store.MagnetFile, parent *store.MagnetFile, idx int) ([]store.MagnetFile, error) {
	if result == nil {
		result = []store.MagnetFile{}
	}

	params := &ListFoldersParams{Id: folderId}
	params.APIKey = apiKey
	c_res, err := c.client.ListFolders(params)
	if err != nil {
		return nil, err
	}

	for _, f := range c_res.Data.Content {
		file := &store.MagnetFile{
			Idx:  idx,
			Name: f.Name,
			Path: "/" + f.Name,
			Size: f.Size,
			Link: f.Link,
		}

		if f.StreamLink != "" {
			file.Link = f.StreamLink
		}

		if parent != nil {
			file.Path = path.Join(parent.Path, file.Name)
		}

		if f.Type == FolderItemTypeFolder {
			result = append(result, *file)
			idx++
			result, err = listFolderFlat(c, apiKey, f.Id, result, file, idx)
			if err != nil {
				return nil, err
			}
		} else {
			result = append(result, *file)
			idx++
		}
	}

	return result, nil
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}

	cm_res, err := c.checkMagnet(&store.CheckMagnetParams{
		Ctx:     params.Ctx,
		Magnets: []string{magnet.Link},
	}, true)
	if err != nil {
		return nil, err
	}

	cm := cm_res.Items[0]

	// already cached, no need to download
	if cm.Status == store.MagnetStatusCached {
		id := CachedMagnetId("").toId(magnet.Hash).toString()

		if _, err = c.ensureFolder(params.APIKey, id); err != nil {
			return nil, err
		}

		data := &store.AddMagnetData{
			Id:     id,
			Hash:   magnet.Hash,
			Magnet: magnet.Link,
			Name:   magnet.Name,
			Status: store.MagnetStatusDownloaded,
			Files:  cm.Files,
		}

		c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

		return data, nil
	}

	// not cached, need to download
	folder, err := c.ensureFolder(params.APIKey, magnet.Hash)
	if err != nil {
		return nil, err
	}

	ct_res, err := c.client.CreateTransfer(&CreateTransferParams{
		Ctx:      params.Ctx,
		Src:      magnet.Link,
		FolderId: folder.Id,
	})
	if err != nil {
		return nil, err
	}

	data := &store.AddMagnetData{
		Id:     ct_res.Data.Id,
		Hash:   magnet.Hash,
		Magnet: magnet.Link,
		Name:   ct_res.Data.Name,
		Status: store.MagnetStatusQueued,
	}

	transfer, err := getTransferById(c, params.APIKey, data.Id)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

		return data, nil
	}

	if transfer.Status == TransferStatusFinished {
		data.Status = store.MagnetStatusDownloaded

		files, err := listFolderFlat(c, params.APIKey, folder.Id, nil, nil, 0)
		if err != nil {
			return nil, err
		}

		data.Files = files
	}

	c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

	return data, nil
}

func getMagnetStatsForTransfer(transfer *ListTransfersDataItem) store.MagnetStatus {
	if transfer.Status == TransferStatusFinished {
		return store.MagnetStatusDownloaded
	}
	if transfer.Status == TransferStatusRunning {
		if transfer.Progress > 0 {
			return store.MagnetStatusDownloading
		}
		return store.MagnetStatusQueued
	}
	return store.MagnetStatusUnknown
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	if CachedMagnetId(params.Id).isValid() {
		magnet, err := core.ParseMagnetLink(CachedMagnetId(params.Id).toHash())
		if err != nil {
			return nil, err
		}
		files, err := c.getCachedMagnetFiles(params.APIKey, magnet.Link, true)
		if err != nil {
			return nil, err
		}
		data := &store.GetMagnetData{
			Id:     params.Id,
			Hash:   magnet.Hash,
			Name:   "",
			Status: store.MagnetStatusDownloaded,
			Files:  files,
		}

		return data, nil
	}

	transfer, err := getTransferById(c, params.APIKey, params.Id)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		err := core.NewAPIError("not found")
		err.StatusCode = http.StatusNotFound
		err.StoreName = string(store.StoreNamePremiumize)
		return nil, err
	}

	magnet, err := core.ParseMagnetLink(transfer.Src)
	if err != nil {
		return nil, err
	}
	data := &store.GetMagnetData{
		Id:     transfer.Id,
		Hash:   magnet.Hash,
		Name:   transfer.Name,
		Status: getMagnetStatsForTransfer(transfer),
	}

	if transfer.Status == TransferStatusFinished {
		files, err := listFolderFlat(c, params.APIKey, transfer.FolderId, nil, &store.MagnetFile{
			Path: "/" + transfer.Name,
		}, 0)
		if err != nil {
			return nil, err
		}
		data.Files = files
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	lm := []store.ListMagnetsDataItem{}
	if !c.listMagnetsCache.Get(c.getCacheKey(params, ""), &lm) {
		sf_res, err := c.client.SearchFolders(&SearchFoldersParams{
			Ctx:   params.Ctx,
			Query: CachedMagnetIdPrefix,
		})
		if err != nil {
			return nil, err
		}

		lt_res, err := c.client.ListTransfers(&ListTransfersParams{
			Ctx: params.Ctx,
		})
		if err != nil {
			return nil, err
		}

		items := []store.ListMagnetsDataItem{}

		for _, m := range sf_res.Data.Content {
			item := &store.ListMagnetsDataItem{
				Id:     m.Name,
				Hash:   CachedMagnetId(m.Name).toHash(),
				Name:   "",
				Status: store.MagnetStatusDownloaded,
			}

			items = append(items, *item)
		}

		for _, t := range lt_res.Data.Transfers {
			magnet, err := core.ParseMagnetLink(t.Src)
			if err != nil {
				return nil, err
			}
			item := &store.ListMagnetsDataItem{
				Id:     t.Id,
				Hash:   magnet.Hash,
				Name:   t.Name,
				Status: getMagnetStatsForTransfer(&t),
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

func (c *StoreClient) deleteTransferById(apiKey string, transferId string) error {
	transfer, err := getTransferById(c, apiKey, transferId)
	if err != nil {
		return err
	}
	if transfer == nil {
		return nil
	}

	dt_params := &DeleteTransferParams{Id: transferId}
	dt_params.APIKey = apiKey
	_, err = c.client.DeleteTransfer(dt_params)
	if err != nil {
		return err
	}

	return nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	if CachedMagnetId(params.Id).isValid() {
		folder, err := c.getFolderByName(params.APIKey, CachedMagnetId(params.Id).toString())
		if err != nil {
			return nil, err
		}

		data := &store.RemoveMagnetData{Id: params.Id}

		if folder == nil {
			return data, nil
		}

		_, err = c.client.DeleteFolder(&DeleteFolderParams{
			Ctx: params.Ctx,
			Id:  folder.Id,
		})
		if err != nil {
			return nil, err
		}

		c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

		return data, nil
	}

	err := c.deleteTransferById(params.APIKey, params.Id)
	if err != nil {
		return nil, err
	}

	c.listMagnetsCache.Remove(c.getCacheKey(params, ""))

	data := &store.RemoveMagnetData{Id: params.Id}
	return data, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	data := &store.GenerateLinkData{Link: params.Link}
	return data, nil
}
