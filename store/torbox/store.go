package torbox

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
}

func NewStoreClient() *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{})
	c.Name = store.StoreNameTorBox
	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
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
	return data, nil
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	magnetByHash := map[string]core.MagnetLink{}
	hashes := []string{}
	for _, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		hash := strings.ToLower(magnet.Hash)
		magnetByHash[hash] = magnet
		hashes = append(hashes, hash)
	}
	res, err := c.client.CheckTorrentsCached(&CheckTorrentsCachedParams{
		Ctx:       params.Ctx,
		Hashes:    hashes,
		ListFiles: true,
	})
	if err != nil {
		return nil, err
	}
	tByHash := map[string]CheckTorrentsCachedDataItem{}
	for _, t := range res.Data {
		tByHash[strings.ToLower(t.Hash)] = t
	}
	data := &store.CheckMagnetData{}
	for _, hash := range hashes {
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
					Name: f.Name,
					Size: f.Size,
				})
			}
		}
		data.Items = append(data.Items, item)
	}
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

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	torrentId, fileId, err := LockedFileLink(params.Link).parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
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
	return data, nil
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	id, err := strconv.Atoi(params.Id)
	if err != nil {
		return nil, err
	}
	res, err := c.client.GetTorrent(&GetTorrentParams{
		Ctx:         params.Ctx,
		Id:          id,
		BypassCache: true,
	})
	if res.Data.Id == 0 {
		error := core.NewAPIError("not found")
		error.StatusCode = http.StatusNotFound
		error.StoreName = string(store.StoreNameTorBox)
		return nil, error
	}
	data := &store.GetMagnetData{
		Id:     params.Id,
		Name:   res.Data.Name,
		Status: store.MagnetStatusQueued,
		Files:  []store.MagnetFile{},
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
	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	res, err := c.client.ListTorrents(&ListTorrentsParams{
		Ctx:         params.Ctx,
		BypassCache: true,
		Offset:      0,
		Limit:       0,
	})
	if err != nil {
		return nil, err
	}
	data := &store.ListMagnetsData{}
	for _, t := range res.Data {
		item := store.ListMagnetsDataItem{
			Id:     strconv.Itoa(t.Id),
			Name:   t.Name,
			Status: store.MagnetStatusUnknown,
		}
		if t.DownloadFinished && t.DownloadPresent {
			item.Status = store.MagnetStatusDownloaded
		}
		data.Items = append(data.Items, item)
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
