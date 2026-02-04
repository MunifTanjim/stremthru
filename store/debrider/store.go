package debrider

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

func getMagnetStatusFromTaskStatus(status TaskStatus) store.MagnetStatus {
	switch status {
	case TaskStatusError:
		return store.MagnetStatusFailed
	case TaskStatusParsing:
		return store.MagnetStatusQueued
	case TaskStatusDownloading:
		return store.MagnetStatusDownloading
	case TaskStatusCompleted:
		return store.MagnetStatusDownloaded
	default:
		return store.MagnetStatusUnknown
	}
}

type StoreClientConfig struct {
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
	config *StoreClientConfig

	generateLinkCache cache.Cache[store.GenerateLinkData]
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/debrider/"

func (l LockedFileLink) encodeData(taskId, fileName string) string {
	return util.Base64Encode(taskId + ":" + fileName)
}

func (l LockedFileLink) decodeData(encoded string) (taskId, fileName string, err error) {
	decoded, err := util.Base64Decode(encoded)
	if err != nil {
		return "", "", err
	}
	tId, fName, found := strings.Cut(decoded, ":")
	if !found {
		return "", "", err
	}
	return tId, fName, nil
}

func (l LockedFileLink) Create(taskId, fileName string) string {
	return lockedFileLinkPrefix + l.encodeData(taskId, fileName)
}

func (l LockedFileLink) Parse() (taskId, fileName string, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	var magnet *core.MagnetLink
	var isPrivate bool
	if params.Magnet != "" {
		m, err := core.ParseMagnetLink(params.Magnet)
		if err != nil {
			return nil, err
		}
		magnet = &m
	} else {
		mi, mii, err := params.GetTorrentMeta()
		if err != nil {
			return nil, err
		}
		isPrivate = util.PtrToBool(mii.Private, false)
		m, err := core.ParseMagnetLink(mi.HashInfoBytes().HexString())
		if err != nil {
			return nil, err
		}
		magnet = &m
	}

	cdt_params := &CreateDownloadTaskParams{
		Ctx: params.Ctx,
	}
	if params.Magnet != "" {
		cdt_params.Type = DownloadTaskTypeMagnet
		cdt_params.Data = CreateDownloadTaskParamsData{
			MagnetLink: magnet.RawLink,
		}
	} else {
		cdt_params.Type = DownloadTaskTypeTorrent
		cdt_params.Data = CreateDownloadTaskParamsData{
			FileContent: params.Torrent,
		}
	}
	res, err := s.client.CreateDownloadTask(cdt_params)
	if err != nil {
		return nil, err
	}
	data := &store.AddMagnetData{
		Id:      res.Data.Id,
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    res.Data.Name,
		Size:    res.Data.Size,
		Status:  getMagnetStatusFromTaskStatus(res.Data.Status),
		Private: isPrivate,
		Files:   []store.MagnetFile{},
		AddedAt: res.Data.GetAddedAt(),
	}
	source := string(s.GetName().Code())
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		data.Files = append(data.Files, store.MagnetFile{
			Idx:    i,
			Link:   LockedFileLink("").Create(res.Data.Id, f.Name),
			Name:   f.GetName(),
			Path:   f.GetPath(),
			Size:   f.Size,
			Source: source,
		})
	}
	return data, nil

}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	totalMagnets := len(params.Magnets)

	magnetByHash := make(map[string]core.MagnetLink, totalMagnets)
	hashes := make([]string, totalMagnets)

	for i, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		magnetByHash[magnet.Hash] = magnet
		hashes[i] = magnet.Hash
	}

	foundItemByHash := map[string]store.CheckMagnetDataItem{}

	if data, err := buddy.CheckMagnet(s, hashes, params.GetAPIKey(s.client.apiKey), params.ClientIP, params.SId); err != nil {
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

	missingHashes := []string{}
	missingLinks := []string{}
	for _, hash := range hashes {
		if _, ok := foundItemByHash[hash]; !ok {
			magnet := magnetByHash[hash]
			missingHashes = append(missingHashes, magnet.Hash)
			missingLinks = append(missingLinks, magnet.Link)
		}
	}

	ldByHash := map[string]CheckLinkAvailabilityDataItem{}
	if len(missingHashes) > 0 {
		res, err := s.client.CheckLinkAvailability(&CheckLinkAvailabilityParams{
			Ctx:  params.Ctx,
			Data: missingLinks,
		})
		if err != nil {
			return nil, err
		}
		for i, detail := range res.Data.Result {
			hash := missingHashes[i]
			ldByHash[hash] = detail
		}
	}
	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}
	tInfos := []buddy.TorrentInfoInput{}
	source := string(s.GetName().Code())
	for _, hash := range hashes {
		if item, ok := foundItemByHash[hash]; ok {
			data.Items = append(data.Items, item)
			continue
		}

		magnet := magnetByHash[hash]
		item := store.CheckMagnetDataItem{
			Hash:   magnet.Hash,
			Magnet: magnet.Link,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}
		tInfo := buddy.TorrentInfoInput{
			Hash:  hash,
			Files: []torrent_stream.File{},
		}
		if detail, ok := ldByHash[hash]; ok {
			if detail.Cached {
				item.Status = store.MagnetStatusCached
				for idx, f := range detail.Files {
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
		}
		data.Items = append(data.Items, item)
		tInfos = append(tInfos, tInfo)
	}
	go buddy.BulkTrackMagnet(s, tInfos, nil, "", params.GetAPIKey(s.client.apiKey))
	return data, nil
}

func (s *StoreClient) getCachedGeneratedLink(params *store.GenerateLinkParams, taskId, fileName string) *store.GenerateLinkData {
	v := &store.GenerateLinkData{}
	if s.generateLinkCache.Get(params.GetAPIKey(s.client.apiKey)+":"+taskId+":"+fileName, v) {
		return v
	}
	return nil

}

func (s *StoreClient) setCachedGenerateLink(params *store.GenerateLinkParams, taskId, fileName string, v *store.GenerateLinkData) {
	s.generateLinkCache.Add(params.GetAPIKey(s.client.apiKey)+":"+taskId+":"+fileName, *v)
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	taskId, fileName, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StoreName = string(store.StoreNameDebrider)
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	if v := s.getCachedGeneratedLink(params, taskId, fileName); v != nil {
		return v, nil
	}
	res, err := s.client.GetTask(&GetTaskParams{
		Ctx: params.Ctx,
		Id:  taskId,
	})
	if err != nil {
		return nil, err
	}
	link := ""
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		if f.Name == fileName {
			link = f.DownloadLink
		}
	}
	if link == "" {
		err := core.NewAPIError("file not found")
		err.StoreName = string(store.StoreNameDebrider)
		err.StatusCode = http.StatusNotFound
		return nil, err
	}
	data := &store.GenerateLinkData{Link: link}
	s.setCachedGenerateLink(params, taskId, fileName, data)
	return data, nil
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	res, err := s.client.GetTask(&GetTaskParams{
		Ctx: params.Ctx,
		Id:  params.Id,
	})
	if err != nil {
		return nil, err
	}
	if res.Data.Type != "torrent" {
		err := core.NewAPIError("not found")
		err.StatusCode = 404
		err.StoreName = string(store.StoreNameDebrider)
		return nil, err
	}
	data := &store.GetMagnetData{
		Id:      res.Data.Id,
		Hash:    res.Data.Hash,
		Name:    res.Data.Name,
		Size:    res.Data.Size,
		Status:  getMagnetStatusFromTaskStatus(res.Data.Status),
		Files:   []store.MagnetFile{},
		AddedAt: res.Data.GetAddedAt(),
	}
	source := string(s.GetName().Code())
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		data.Files = append(data.Files, store.MagnetFile{
			Idx:    i,
			Link:   LockedFileLink("").Create(res.Data.Id, f.Name),
			Name:   f.GetName(),
			Path:   f.GetPath(),
			Size:   f.Size,
			Source: source,
		})
	}
	return data, nil
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := s.client.GetAccount(&GetAccountParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:                 res.Data.Id,
		Email:              res.Data.Email,
		SubscriptionStatus: store.UserSubscriptionStatusExpired,
	}
	switch res.Data.Subscription.Status {
	case "active":
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	case "trialing":
		data.SubscriptionStatus = store.UserSubscriptionStatusTrial
	case "canceled":
		data.SubscriptionStatus = store.UserSubscriptionStatusExpired
	}
	return data, err
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	res, err := s.client.ListTask(&ListTaskParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}

	items := []store.ListMagnetsDataItem{}
	for i := range res.Data {
		task := &res.Data[i]
		if task.Type != "torrent" {
			continue
		}
		item := store.ListMagnetsDataItem{
			Id:      task.Id,
			Hash:    task.Hash,
			Name:    task.Name,
			Size:    task.Size,
			Status:  getMagnetStatusFromTaskStatus(task.Status),
			AddedAt: task.GetAddedAt(),
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
	_, err := s.client.DeleteTask(&DeleteTaskParams{
		Ctx: params.Ctx,
		Id:  params.Id,
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
	c.Name = store.StoreNameDebrider
	c.config = config

	c.generateLinkCache = func() cache.Cache[store.GenerateLinkData] {
		return cache.NewCache[store.GenerateLinkData](&cache.CacheConfig{
			Name:     "store:debrider:generateLink",
			Lifetime: 30 * time.Minute,
		})
	}()

	return c
}
