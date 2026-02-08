package qbittorrent

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/store"
)

type StoreClientConfig struct {
	HTTPClient *http.Client
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		HTTPClient: config.HTTPClient,
	})
	c.Name = store.StoreNameQBittorrent
	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) getConfig(apiKey string) (*qbitConfig, error) {
	if apiKey == "" {
		err := core.NewStoreError("missing api key")
		err.StoreName = string(store.StoreNameQBittorrent)
		err.StatusCode = http.StatusUnauthorized
		return nil, err
	}
	return parseToken(apiKey)
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/qbittorrent/"

func (l LockedFileLink) encodeData(hash string, fileIndex int) string {
	return core.Base64Encode(hash + ":" + strconv.Itoa(fileIndex))
}

func (l LockedFileLink) decodeData(encoded string) (hash string, fileIndex int, err error) {
	decoded, err := core.Base64Decode(encoded)
	if err != nil {
		return "", 0, err
	}
	h, idx, found := strings.Cut(decoded, ":")
	if !found {
		return "", 0, fmt.Errorf("invalid locked file link data")
	}
	fileIndex, err = strconv.Atoi(idx)
	if err != nil {
		return "", 0, err
	}
	return h, fileIndex, nil
}

func (l LockedFileLink) create(hash string, fileIndex int) string {
	return lockedFileLinkPrefix + l.encodeData(hash, fileIndex)
}

func (l LockedFileLink) parse() (hash string, fileIndex int, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func progressToStatus(progress float64) store.MagnetStatus {
	if progress >= 1.0 {
		return store.MagnetStatusDownloaded
	} else if progress > 0 {
		return store.MagnetStatusDownloading
	}
	return store.MagnetStatusQueued
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	_, err = c.client.GetVersion(cfg)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	data := &store.User{
		Id:                 cfg.Username + "@" + cfg.URL,
		Email:              "",
		SubscriptionStatus: store.UserSubscriptionStatusPremium,
	}

	return data, nil
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	var magnet *core.MagnetLink
	if params.Magnet != "" {
		m, err := core.ParseMagnetLink(params.Magnet)
		if err != nil {
			return nil, err
		}
		magnet = &m
	} else if params.Torrent != nil {
		mi, _, err := params.GetTorrentMeta()
		if err != nil {
			return nil, err
		}
		m, err := core.ParseMagnetLink(mi.HashInfoBytes().HexString())
		if err != nil {
			return nil, err
		}
		magnet = &m
	} else {
		return nil, fmt.Errorf("either magnet or torrent must be provided")
	}

	err = c.client.AddTorrentMagnet(cfg, magnet.RawLink)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	// Poll briefly for torrent metadata to become available
	var torrent *TorrentInfo
	for i := 0; i < 5; i++ {
		time.Sleep(500 * time.Millisecond)
		torrents, err := c.client.GetTorrents(cfg, []string{magnet.Hash}, 0, 0)
		if err == nil && len(torrents) > 0 {
			torrent = &torrents[0]
			break
		}
	}

	data := &store.AddMagnetData{
		Id:      magnet.Hash,
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    magnet.Name,
		Status:  store.MagnetStatusQueued,
		Files:   []store.MagnetFile{},
		AddedAt: time.Now().UTC(),
	}

	if torrent != nil {
		data.Name = torrent.Name
		data.Size = torrent.TotalSize
		data.Status = progressToStatus(torrent.Progress)
		data.AddedAt = torrent.GetAddedAt()

		files, err := c.client.GetTorrentFiles(cfg, magnet.Hash)
		if err == nil {
			source := string(c.GetName().Code())
			for _, f := range files {
				data.Files = append(data.Files, store.MagnetFile{
					Idx:    f.Index,
					Link:   LockedFileLink("").create(magnet.Hash, f.Index),
					Name:   f.GetName(),
					Path:   f.GetPath(),
					Size:   f.Size,
					Source: source,
				})
			}
		}
	}

	return data, nil
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

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

	if data, err := buddy.CheckMagnet(c, hashes, params.GetAPIKey(""), params.ClientIP, params.SId); err != nil {
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
	for _, hash := range hashes {
		if _, ok := foundItemByHash[hash]; !ok {
			missingHashes = append(missingHashes, hash)
		}
	}

	torrentByHash := map[string]TorrentInfo{}
	if len(missingHashes) > 0 {
		torrents, err := c.client.GetTorrents(cfg, missingHashes, 0, 0)
		if err != nil {
			return nil, UpstreamErrorWithCause(err)
		}
		for _, t := range torrents {
			torrentByHash[strings.ToLower(t.Hash)] = t
		}
	}

	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}
	tInfos := []buddy.TorrentInfoInput{}
	source := string(c.GetName().Code())

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
			Hash:  hash,
			Files: torrent_stream.Files{},
		}

		if t, ok := torrentByHash[hash]; ok {
			tInfo.TorrentTitle = t.Name
			tInfo.Size = t.TotalSize

			if t.Progress >= 1.0 {
				item.Status = store.MagnetStatusCached
				files, err := c.client.GetTorrentFiles(cfg, hash)
				if err == nil {
					for _, f := range files {
						file := torrent_stream.File{
							Idx:    f.Index,
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
			} else if t.Progress > 0 {
				item.Status = store.MagnetStatusDownloading
			} else {
				item.Status = store.MagnetStatusQueued
			}
		}

		data.Items = append(data.Items, item)
		tInfos = append(tInfos, tInfo)
	}

	go buddy.BulkTrackMagnet(c, tInfos, nil, "", params.GetAPIKey(""))

	return data, nil
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	hash := strings.ToLower(params.Id)

	torrents, err := c.client.GetTorrents(cfg, []string{hash}, 0, 0)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}
	if len(torrents) == 0 {
		apiErr := core.NewAPIError("torrent not found")
		apiErr.StatusCode = http.StatusNotFound
		apiErr.StoreName = string(store.StoreNameQBittorrent)
		return nil, apiErr
	}

	t := torrents[0]
	data := &store.GetMagnetData{
		Id:      t.Hash,
		Hash:    strings.ToLower(t.Hash),
		Name:    t.Name,
		Size:    t.TotalSize,
		Status:  progressToStatus(t.Progress),
		Files:   []store.MagnetFile{},
		AddedAt: t.GetAddedAt(),
	}

	files, err := c.client.GetTorrentFiles(cfg, hash)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	source := string(c.GetName().Code())
	for _, f := range files {
		data.Files = append(data.Files, store.MagnetFile{
			Idx:    f.Index,
			Link:   LockedFileLink("").create(hash, f.Index),
			Name:   f.GetName(),
			Path:   f.GetPath(),
			Size:   f.Size,
			Source: source,
		})
	}

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	torrents, err := c.client.GetTorrents(cfg, nil, limit, params.Offset)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	data := &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: 0,
	}

	for _, t := range torrents {
		item := store.ListMagnetsDataItem{
			Id:      t.Hash,
			Hash:    strings.ToLower(t.Hash),
			Name:    t.Name,
			Size:    t.TotalSize,
			Status:  progressToStatus(t.Progress),
			AddedAt: t.GetAddedAt(),
		}
		data.Items = append(data.Items, item)
	}

	count := len(data.Items)
	data.TotalItems = params.Offset + count
	if count == limit {
		data.TotalItems += 1
	}

	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	hash := strings.ToLower(params.Id)

	err = c.client.DeleteTorrents(cfg, []string{hash}, false)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	return &store.RemoveMagnetData{Id: params.Id}, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	cfg, err := c.getConfig(params.GetAPIKey(""))
	if err != nil {
		return nil, err
	}

	hash, fileIndex, err := LockedFileLink(params.Link).parse()
	if err != nil {
		apiErr := core.NewAPIError("invalid link")
		apiErr.StatusCode = http.StatusBadRequest
		apiErr.Cause = err
		return nil, apiErr
	}

	files, err := c.client.GetTorrentFiles(cfg, hash)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	if fileIndex < 0 || fileIndex >= len(files) {
		apiErr := core.NewAPIError("file index out of range")
		apiErr.StatusCode = http.StatusBadRequest
		return nil, apiErr
	}

	file := files[fileIndex]

	// URL-encode each path segment individually
	parts := strings.Split(file.Name, "/")
	encodedParts := make([]string, len(parts))
	for i, p := range parts {
		encodedParts[i] = url.PathEscape(p)
	}
	encodedPath := strings.Join(encodedParts, "/")

	link := cfg.FileBaseURL + "/" + encodedPath

	return &store.GenerateLinkData{Link: link}, nil
}
