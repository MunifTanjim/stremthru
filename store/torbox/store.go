package torbox

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	Name              store.StoreName
	client            *APIClient
	getUserCache      cache.Cache[store.User]
	getMagnetCache    cache.Cache[store.GetMagnetData] // for downloaded magnets
	generateLinkCache cache.Cache[store.GenerateLinkData]
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameTorBox

	c.getUserCache = func() cache.Cache[store.User] {
		return cache.NewCache[store.User](&cache.CacheConfig{
			Name:     "store:torbox:getUser",
			Lifetime: 1 * time.Minute,
		})
	}()

	c.getMagnetCache = func() cache.Cache[store.GetMagnetData] {
		return cache.NewCache[store.GetMagnetData](&cache.CacheConfig{
			Name:     "store:torbox:getMagnet",
			Lifetime: 10 * time.Minute,
		})
	}()

	c.generateLinkCache = func() cache.Cache[store.GenerateLinkData] {
		return cache.NewCache[store.GenerateLinkData](&cache.CacheConfig{
			Name:     "store:torbox:generateLink",
			Lifetime: 50 * time.Minute,
		})
	}()

	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) getCachedGetUser(params *store.GetUserParams) *store.User {
	v := &store.User{}
	if c.getUserCache.Get(params.GetAPIKey(c.client.apiKey), v) {
		return v
	}
	return nil
}

func (c *StoreClient) setCachedGetUser(params *store.GetUserParams, v *store.User) {
	c.getUserCache.Add(params.GetAPIKey(c.client.apiKey), *v)
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	if v := c.getCachedGetUser(params); v != nil {
		return v, nil
	}
	res, err := c.client.GetUser(&GetUserParams{
		Ctx:      params.Ctx,
		Settings: true,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:    strconv.Itoa(res.Data.Id),
		Email: res.Data.Email,
	}
	if res.Data.Plan == PlanFree {
		data.SubscriptionStatus = store.UserSubscriptionStatusTrial
	} else {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	}
	c.setCachedGetUser(params, data)
	return data, nil
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	magnetByHash := make(map[string]core.MagnetLink, len(params.Magnets))
	hashes := make([]string, len(params.Magnets))

	missingHashes := []string{}

	for i, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		magnetByHash[magnet.Hash] = magnet
		hashes[i] = magnet.Hash
	}

	foundItemByHash := map[string]store.CheckMagnetDataItem{}

	if data, err := buddy.CheckMagnet(c, hashes, params.GetAPIKey(c.client.apiKey), params.ClientIP, params.SId); err != nil {
		return nil, err
	} else {
		for _, item := range data.Items {
			foundItemByHash[item.Hash] = item
		}
	}

	if params.LocalOnly {
		data := &store.CheckMagnetData{
			Items: []store.CheckMagnetDataItem{},
		}

		for _, hash := range hashes {
			if item, ok := foundItemByHash[hash]; ok {
				data.Items = append(data.Items, item)
			}
		}
		return data, nil
	}

	for _, hash := range hashes {
		if _, ok := foundItemByHash[hash]; !ok {
			missingHashes = append(missingHashes, hash)
		}
	}

	tByHash := map[string]CheckTorrentsCachedDataItem{}
	if len(missingHashes) > 0 {
		chunkCount := len(missingHashes)/100 + 1
		cItems := make([][]CheckTorrentsCachedDataItem, chunkCount)
		errs := make([]error, chunkCount)
		hasError := false

		var wg sync.WaitGroup
		for i, cMissingHashes := range slices.Collect(slices.Chunk(missingHashes, 100)) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctcParams := &CheckTorrentsCachedParams{
					Hashes:    cMissingHashes,
					ListFiles: true,
				}
				ctcParams.APIKey = params.APIKey
				res, err := c.client.CheckTorrentsCached(ctcParams)
				if err != nil {
					hasError = true
					errs[i] = err
					return
				}
				cItems[i] = res.Data
			}()
		}
		wg.Wait()

		if hasError {
			return nil, errors.Join(errs...)
		}

		for _, items := range cItems {
			for _, t := range items {
				tByHash[strings.ToLower(t.Hash)] = t
			}
		}
	}
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}
	tInfos := []buddy.TorrentInfoInput{}
	filesByHash := map[string][]store.MagnetFile{}
	for _, hash := range hashes {
		if item, ok := foundItemByHash[hash]; ok {
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
			item.Status = store.MagnetStatusCached
			for idx, f := range t.Files {
				item.Files = append(item.Files, store.MagnetFile{
					Idx:  idx,
					Name: f.GetName(),
					Size: f.Size,
				})
			}
			tInfos = append(tInfos, buddy.TorrentInfoInput{
				Hash:         hash,
				TorrentTitle: t.Name,
				Size:         t.Size,
			})
		}
		data.Items = append(data.Items, item)
		filesByHash[hash] = item.Files
	}
	go buddy.BulkTrackMagnet(c, tInfos, filesByHash, "", params.GetAPIKey(c.client.apiKey))
	return data, nil
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/torbox/"

func (l LockedFileLink) encodeData(torrentId int, torrentFileId int) string {
	return core.Base64Encode(strconv.Itoa(torrentId) + ":" + strconv.Itoa(torrentFileId))
}

func (l LockedFileLink) decodeData(encoded string) (torrentId, torrentFileId int, err error) {
	decoded, err := core.Base64Decode(encoded)
	if err != nil {
		return 0, 0, err
	}
	tId, tfId, found := strings.Cut(decoded, ":")
	if !found {
		return 0, 0, err
	}
	torrentId, err = strconv.Atoi(tId)
	if err != nil {
		return 0, 0, err
	}
	torrentFileId, err = strconv.Atoi(tfId)
	if err != nil {
		return 0, 0, err
	}
	return torrentId, torrentFileId, nil
}

func (l LockedFileLink) create(torrentId int, torrentFileId int) string {
	return lockedFileLinkPrefix + l.encodeData(torrentId, torrentFileId)
}

func (l LockedFileLink) parse() (torrentId, torrentFileId int, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}
	res, err := c.client.CreateTorrent(&CreateTorrentParams{
		Ctx:      params.Ctx,
		Magnet:   magnet.Link,
		AllowZip: false,
	})
	if err != nil {
		return nil, err
	}
	data := &store.AddMagnetData{
		Id:     strconv.Itoa(res.Data.TorrentId),
		Hash:   res.Data.Hash,
		Magnet: magnet.Link,
		Name:   res.Data.Name,
		Status: store.MagnetStatusQueued,
		Files:  []store.MagnetFile{},
	}
	t, err := c.client.GetTorrent(&GetTorrentParams{
		Ctx:         params.Ctx,
		Id:          res.Data.TorrentId,
		BypassCache: true,
	})
	if err != nil {
		return nil, err
	}
	data.Size = t.Data.Size
	data.AddedAt = t.Data.GetAddedAt()
	if t.Data.DownloadFinished && t.Data.DownloadPresent {
		data.Status = store.MagnetStatusDownloaded
	}
	for _, f := range t.Data.Files {
		file := store.MagnetFile{
			Idx:  f.Id,
			Link: LockedFileLink("").create(res.Data.TorrentId, f.Id),
			Name: f.ShortName,
			Path: "/" + f.Name,
			Size: f.Size,
		}
		data.Files = append(data.Files, file)
	}

	return data, nil
}

func intToStr(key ...int) string {
	str := ""
	for _, k := range key {
		str = str + ":" + strconv.Itoa(k)
	}
	return str

}

func (c *StoreClient) getCachedGeneratedLink(params *store.GenerateLinkParams, torrentId int, fileId int) *store.GenerateLinkData {
	v := &store.GenerateLinkData{}
	if c.generateLinkCache.Get(params.GetAPIKey(c.client.apiKey)+":"+intToStr(torrentId, fileId), v) {
		return v
	}
	return nil

}

func (c *StoreClient) setCachedGenerateLink(params *store.GenerateLinkParams, torrentId int, fileId int, v *store.GenerateLinkData) {
	c.generateLinkCache.Add(params.GetAPIKey(c.client.apiKey)+":"+intToStr(torrentId, fileId), *v)
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	torrentId, fileId, err := LockedFileLink(params.Link).parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	if v := c.getCachedGeneratedLink(params, torrentId, fileId); v != nil {
		return v, nil
	}
	res, err := c.client.RequestDownloadLink(&RequestDownloadLinkParams{
		Ctx:       params.Ctx,
		TorrentId: torrentId,
		FileId:    fileId,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GenerateLinkData{Link: res.Data.Link}
	c.setCachedGenerateLink(params, torrentId, fileId, data)
	return data, nil
}

func (c *StoreClient) getCachedGetMagnet(params *store.GetMagnetParams) *store.GetMagnetData {
	v := &store.GetMagnetData{}
	if c.getMagnetCache.Get(params.GetAPIKey(c.client.apiKey)+":"+params.Id, v) {
		return v
	}
	return nil
}

func (c *StoreClient) setCachedGetMagnet(params *store.GetMagnetParams, v *store.GetMagnetData) {
	c.getMagnetCache.Add(params.GetAPIKey(c.client.apiKey)+":"+params.Id, *v)
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	if v := c.getCachedGetMagnet(params); v != nil {
		return v, nil
	}
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		return nil, err
	}
	res, err := c.client.GetTorrent(&GetTorrentParams{
		Ctx:         params.Ctx,
		Id:          id,
		BypassCache: true,
	})
	if err != nil {
		return nil, err
	}
	if res.Data.Id == 0 {
		error := core.NewAPIError("not found")
		error.StatusCode = http.StatusNotFound
		error.StoreName = string(store.StoreNameTorBox)
		return nil, error
	}
	data := &store.GetMagnetData{
		Id:      params.Id,
		Hash:    res.Data.Hash,
		Name:    res.Data.Name,
		Size:    res.Data.Size,
		Status:  store.MagnetStatusQueued,
		Files:   []store.MagnetFile{},
		AddedAt: res.Data.GetAddedAt(),
	}
	if res.Data.DownloadFinished && res.Data.DownloadPresent {
		data.Status = store.MagnetStatusDownloaded
	}
	for _, f := range res.Data.Files {
		file := store.MagnetFile{
			Idx:  f.Id,
			Link: LockedFileLink("").create(res.Data.Id, f.Id),
			Name: f.ShortName,
			Path: "/" + f.Name,
			Size: f.Size,
		}
		data.Files = append(data.Files, file)
	}
	if data.Status == store.MagnetStatusDownloaded {
		c.setCachedGetMagnet(params, data)
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	res, err := c.client.ListTorrents(&ListTorrentsParams{
		Ctx:         params.Ctx,
		BypassCache: true,
		Limit:       params.Limit,
		Offset:      params.Offset,
	})
	if err != nil {
		return nil, err
	}
	data := &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: 0,
	}
	for _, t := range res.Data {
		item := store.ListMagnetsDataItem{
			Id:      strconv.Itoa(t.Id),
			Hash:    t.Hash,
			Name:    t.Name,
			Size:    t.Size,
			Status:  store.MagnetStatusUnknown,
			AddedAt: t.GetAddedAt(),
		}
		if t.DownloadFinished && t.DownloadPresent {
			item.Status = store.MagnetStatusDownloaded
		}
		data.Items = append(data.Items, item)
	}
	// torbox returns 1 extra item
	if len(data.Items) > params.Limit {
		data.Items = data.Items[0:params.Limit]
	}
	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		return nil, err
	}
	_, err = c.client.ControlTorrent(&ControlTorrentParams{
		Ctx:       params.Ctx,
		TorrentId: id,
		Operation: ControlTorrentOperationDelete,
	})
	if err != nil {
		return nil, err
	}
	data := &store.RemoveMagnetData{Id: params.Id}
	return data, nil
}
