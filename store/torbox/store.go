package torbox

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/store"
)

var (
	_ store.Store     = (*StoreClient)(nil)
	_ store.NewzStore = (*StoreClient)(nil)
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
	switch res.Data.Plan {
	case PlanFree:
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	case PlanEssential, PlanStandard:
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	case PlanPro:
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
		data.HasUsenet = true
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
		ctcParams := &CheckTorrentsCachedParams{
			Hashes:    missingHashes,
			ListFiles: true,
		}
		ctcParams.APIKey = params.APIKey
		res, err := c.client.CheckTorrentsCached(ctcParams)
		if err != nil {
			return nil, err
		}
		for _, t := range res.Data {
			tByHash[strings.ToLower(t.Hash)] = t
		}
	}
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}
	tInfos := []buddy.TorrentInfoInput{}
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
		tInfo := buddy.TorrentInfoInput{
			Hash: hash,
		}
		if t, ok := tByHash[hash]; ok {
			tInfo.TorrentTitle = t.GetName()
			tInfo.Size = t.Size
			item.Status = store.MagnetStatusCached
			source := string(c.GetName().Code())
			for idx, f := range t.Files {
				file := torrent_stream.File{
					Idx:    idx,
					Path:   f.GetPath(),
					Name:   f.GetName(),
					Size:   f.Size,
					Source: source,
				}
				tInfo.Files = append(tInfo.Files, file)
				item.Files = append(item.Files, store.MagnetFile{
					Idx:    file.Idx,
					Path:   file.Path,
					Name:   file.Name,
					Size:   file.Size,
					Source: file.Source,
				})
			}
		}
		tInfos = append(tInfos, tInfo)
		data.Items = append(data.Items, item)
	}
	go buddy.BulkTrackMagnet(c, tInfos, nil, "", params.GetAPIKey(c.client.apiKey))
	return data, nil
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/torbox/"

func (l LockedFileLink) encodeData(id int, fileId int) string {
	return core.Base64Encode(strconv.Itoa(id) + ":" + strconv.Itoa(fileId))
}

func (l LockedFileLink) decodeData(encoded string) (id, fileId int, err error) {
	decoded, err := core.Base64Decode(encoded)
	if err != nil {
		return 0, 0, err
	}
	tId, tfId, found := strings.Cut(decoded, ":")
	if !found {
		return 0, 0, err
	}
	id, err = strconv.Atoi(tId)
	if err != nil {
		return 0, 0, err
	}
	fileId, err = strconv.Atoi(tfId)
	if err != nil {
		return 0, 0, err
	}
	return id, fileId, nil
}

func (l LockedFileLink) Create(id int, fileId int) string {
	return lockedFileLinkPrefix + l.encodeData(id, fileId)
}

func (l LockedFileLink) Parse() (id, fileId int, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	ctParams := &CreateTorrentParams{
		Ctx:      params.Ctx,
		AllowZip: false,
		File:     params.Torrent,
	}

	var magnet *core.MagnetLink
	var err error
	if params.Magnet != "" {
		m, err := core.ParseMagnetLink(params.Magnet)
		if err != nil {
			return nil, err
		}
		magnet = &m
		ctParams.Magnet = magnet.RawLink
	} else {
		ctParams.File = params.Torrent
	}

	res, err := c.client.CreateTorrent(ctParams)
	if err != nil {
		return nil, err
	}
	if magnet == nil {
		m, err := core.ParseMagnetLink(res.Data.Hash)
		if err != nil {
			return nil, err
		}
		magnet = &m
	}
	data := &store.AddMagnetData{
		Id:     strconv.Itoa(res.Data.TorrentId),
		Hash:   res.Data.Hash,
		Magnet: magnet.Link,
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
	data.Name = t.Data.GetName()
	data.Size = t.Data.Size
	data.Private = t.Data.Private
	data.AddedAt = t.Data.GetAddedAt()
	if t.Data.DownloadFinished && t.Data.DownloadPresent {
		data.Status = store.MagnetStatusDownloaded
	} else if t.Data.Progress > 0 {
		data.Status = store.MagnetStatusDownloading
	}
	source := string(c.GetName().Code())
	for _, f := range t.Data.Files {
		file := store.MagnetFile{
			Idx:       f.Id,
			Link:      LockedFileLink("").Create(res.Data.TorrentId, f.Id),
			Name:      f.GetName(),
			Path:      f.GetPath(),
			Size:      f.Size,
			VideoHash: f.OpensubtitlesHash,
			Source:    source,
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
	torrentId, fileId, err := LockedFileLink(params.Link).Parse()
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
		UserIP:    params.ClientIP,
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
		Name:    res.Data.GetName(),
		Size:    res.Data.Size,
		Status:  store.MagnetStatusQueued,
		Files:   []store.MagnetFile{},
		Private: res.Data.Private,
		AddedAt: res.Data.GetAddedAt(),
	}
	if res.Data.DownloadFinished && res.Data.DownloadPresent {
		data.Status = store.MagnetStatusDownloaded
	} else if res.Data.Progress > 0 {
		data.Status = store.MagnetStatusDownloading
	}
	source := string(c.GetName().Code())
	for _, f := range res.Data.Files {
		file := store.MagnetFile{
			Idx:       f.Id,
			Link:      LockedFileLink("").Create(res.Data.Id, f.Id),
			Name:      f.GetName(),
			Path:      f.GetPath(),
			Size:      f.Size,
			VideoHash: f.OpensubtitlesHash,
			Source:    source,
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
			Name:    t.GetName(),
			Size:    t.Size,
			Status:  store.MagnetStatusUnknown,
			Private: t.Private,
			AddedAt: t.GetAddedAt(),
		}
		if t.DownloadFinished && t.DownloadPresent {
			item.Status = store.MagnetStatusDownloaded
		} else if t.Progress > 0 {
			item.Status = store.MagnetStatusDownloading
		}
		data.Items = append(data.Items, item)
	}
	count := len(data.Items)
	// torbox returns 1 extra item
	if count > params.Limit {
		data.Items = data.Items[0:params.Limit]
		count = params.Limit
	}
	data.TotalItems = params.Offset + count
	if count == params.Limit {
		data.TotalItems += 1
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

func (c *StoreClient) CheckNewz(params *store.CheckNewzParams) (*store.CheckNewzData, error) {
	cucParams := &CheckUsenetCachedParams{
		Hashes:    params.Hashes,
		ListFiles: true,
	}
	cucParams.APIKey = params.APIKey
	res, err := c.client.CheckUsenetCached(cucParams)
	if err != nil {
		return nil, err
	}

	itemByHash := map[string]CheckUsenetCachedDataItem{}

	for _, item := range res.Data {
		itemByHash[item.Hash] = item
	}

	data := &store.CheckNewzData{
		Items: []store.CheckNewzDataItem{},
	}

	for _, hash := range params.Hashes {
		item := store.CheckNewzDataItem{
			Hash:   hash,
			Status: store.NewzStatusUnknown,
			Files:  []store.NewzFile{},
		}
		if n, ok := itemByHash[hash]; ok {
			item.Status = store.NewzStatusCached
			for idx, f := range n.Files {
				file := store.NewzFile{
					Idx:       idx,
					Path:      f.GetPath(),
					Name:      f.GetName(),
					Size:      f.Size,
					VideoHash: f.OpenSubtitlesHash,
				}
				item.Files = append(item.Files, file)
			}
		}
		data.Items = append(data.Items, item)
	}
	return data, nil
}

func (c *StoreClient) AddNewz(params *store.AddNewzParams) (*store.AddNewzData, error) {
	rParams := &CreateUsenetDownloadParams{
		Ctx:  params.Ctx,
		Link: params.Link,
		File: params.File,
	}
	res, err := c.client.CreateUsenetDownload(rParams)
	if err != nil {
		return nil, err
	}
	data := &store.AddNewzData{
		Id:     strconv.Itoa(res.Data.UsenetDownloadId),
		Hash:   res.Data.Hash,
		Status: store.NewzStatusQueued,
	}
	return data, nil
}

func (c *StoreClient) GetNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		return nil, err
	}
	rParams := &GetUsenetDownloadParams{
		Ctx:         params.Ctx,
		Id:          id,
		BypassCache: true,
	}
	res, err := c.client.GetUsenetDownload(rParams)
	if err != nil {
		return nil, err
	}
	und := &res.Data
	data := &store.GetNewzData{
		Id:     strconv.Itoa(und.Id),
		Hash:   und.Hash,
		Name:   und.Name,
		Size:   und.Size,
		Status: store.NewzStatusUnknown,
	}
	if und.DownloadState == TorrentDownloadStateDownloading {
		data.Status = store.NewzStatusDownloading
	}
	if und.DownloadFinished && und.DownloadPresent {
		data.Status = store.NewzStatusDownloaded
	}
	for i := range und.Files {
		f := &und.Files[i]
		file := store.NewzFile{
			Idx:       f.Id,
			Link:      LockedFileLink("").Create(und.Id, f.Id),
			Name:      f.GetName(),
			Path:      f.GetPath(),
			Size:      f.Size,
			VideoHash: f.OpensubtitlesHash,
		}
		data.Files = append(data.Files, file)
	}
	return data, nil
}

func (c *StoreClient) GenerateNewzLink(params *store.GenerateNewzLinkParams) (*store.GenerateNewzLinkData, error) {
	usenetId, fileId, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	res, err := c.client.RequestUsenetDownloadLink(&RequestUsenetDownloadLinkParams{
		Ctx:      params.Ctx,
		UsenetId: usenetId,
		FileId:   fileId,
		UserIP:   params.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	data := &store.GenerateNewzLinkData{Link: res.Data.Link}
	return data, nil
}
