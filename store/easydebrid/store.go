package easydebrid

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClientConfig struct {
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameEasyDebrid

	return c
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/easydebrid/"

func (l LockedFileLink) encodeData(magnetHash string, fileIdx int) string {
	return core.Base64Encode(magnetHash + ":" + strconv.Itoa(fileIdx))
}

func (l LockedFileLink) decodeData(encoded string) (magnetHash string, fileIdx int, err error) {
	decoded, err := core.Base64Decode(encoded)
	if err != nil {
		return "", 0, err
	}
	magnetHash, fIdx, found := strings.Cut(decoded, ":")
	if !found {
		return "", 0, err
	}
	fileIdx, err = strconv.Atoi(fIdx)
	if err != nil {
		return "", 0, err
	}
	return magnetHash, fileIdx, nil
}

func (l LockedFileLink) create(magnetHash string, fileIdx int) string {
	return lockedFileLinkPrefix + l.encodeData(magnetHash, fileIdx)
}

func (l LockedFileLink) parse() (magnetHash string, fileIdx int, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

const magnetIdPrefix = "st:ed:"

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
	res, err := s.client.LookupLinkDetails(&LookupLinkDetailsParams{
		Ctx:  params.Ctx,
		URLs: []string{magnet.RawLink},
	})
	if err != nil {
		return nil, err
	}
	data := &store.AddMagnetData{
		Id:      magnetIdPrefix + magnet.Hash,
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    magnet.Name,
		Status:  store.MagnetStatusUnknown,
		Private: isPrivate,
		Files:   []store.MagnetFile{},
		AddedAt: time.Unix(0, 0),
	}
	if len(res.Data.Result) == 0 {
		return data, nil
	}
	detail := res.Data.Result[0]
	if name := detail.GetName(); name != "" {
		data.Name = name
	}
	if detail.Cached {
		data.Status = store.MagnetStatusDownloaded
		source := string(s.GetName().Code())
		for idx, f := range detail.Files {
			data.Files = append(data.Files, store.MagnetFile{
				Idx:    idx,
				Link:   LockedFileLink("").create(magnet.Hash, idx),
				Name:   f.GetName(),
				Path:   f.GetPath(),
				Size:   f.Size,
				Source: source,
			})
		}
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

	ldByHash := map[string]LookupLinkDetailsDataItem{}
	if len(missingHashes) > 0 {
		res, err := s.client.LookupLinkDetails(&LookupLinkDetailsParams{
			Ctx:  params.Ctx,
			URLs: missingLinks,
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
			if !detail.Cached {
				continue
			}
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
		data.Items = append(data.Items, item)
		tInfos = append(tInfos, tInfo)
	}
	go buddy.BulkTrackMagnet(s, tInfos, nil, "", params.GetAPIKey(s.client.apiKey))
	return data, nil
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	magnetHash, fileIdx, err := LockedFileLink(params.Link).parse()
	if err != nil {
		return nil, err
	}
	magnet, err := core.ParseMagnetLink(magnetHash)
	if err != nil {
		return nil, err
	}
	res, err := s.client.GenerateLink(&GenerateLinkParams{
		Ctx:      params.Ctx,
		URL:      magnet.Link,
		ClientIP: params.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data.Files) < fileIdx+1 {
		err := UpstreamErrorWithCause(&ResponseContainer{
			Err: string(core.ErrorCodeNotFound),
		})
		err.StatusCode = http.StatusNotFound
		return nil, err
	}
	file := res.Data.Files[fileIdx]
	data := &store.GenerateLinkData{
		Link: file.URL,
	}
	return data, nil
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	magnet, err := core.ParseMagnetLink(strings.TrimPrefix(params.Id, magnetIdPrefix))
	if err != nil {
		return nil, err
	}
	res, err := s.client.LookupLinkDetails(&LookupLinkDetailsParams{
		Ctx:  params.Ctx,
		URLs: []string{magnet.Link},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data.Result) < 1 || !res.Data.Result[0].Cached {
		err := UpstreamErrorWithCause(&ResponseContainer{
			Err: string(core.ErrorCodeNotFound),
		})
		err.StatusCode = http.StatusNotFound
		return nil, err
	}
	detail := res.Data.Result[0]
	data := &store.GetMagnetData{
		Id:      params.Id,
		Hash:    magnet.Hash,
		Name:    detail.GetName(),
		Size:    0,
		Status:  store.MagnetStatusDownloaded,
		Files:   []store.MagnetFile{},
		AddedAt: time.Unix(0, 0),
	}
	if data.Name == "" {
		data.Name = magnet.Name
	}
	source := string(s.GetName().Code())
	for idx, f := range detail.Files {
		data.Size += f.Size
		data.Files = append(data.Files, store.MagnetFile{
			Idx:    idx,
			Link:   LockedFileLink("").create(magnet.Hash, idx),
			Name:   f.GetName(),
			Path:   f.GetPath(),
			Size:   f.Size,
			Source: source,
		})
	}
	return data, nil
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := s.client.GetUserDetails(&GetUserDetailsParams{
		Ctx: params.Ctx,
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:                 string(res.Data.Id),
		Email:              "",
		SubscriptionStatus: store.UserSubscriptionStatusExpired,
	}
	if res.Data.PaidUntil > time.Now().Unix() {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	}
	return data, nil
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	return &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: 0,
	}, nil
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	return &store.RemoveMagnetData{
		Id: params.Id,
	}, nil
}
