package webtor

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/stats"
	webtorsdk "github.com/webtor-io/api-sdk-go"
	"golang.org/x/sync/errgroup"
)

// Webtor (https://webtor.io) streams torrents on demand instead of
// downloading them into cloud storage first, so a stored torrent is
// immediately playable: AddMagnet/GetMagnet report MagnetStatusDownloaded as
// soon as the metainfo is known, and GenerateLink resolves a fresh
// short-lived download URL on every call. The client is the official Go SDK:
// https://github.com/webtor-io/api-sdk-go

const checkMagnetConcurrency = 8

type StoreClientConfig struct {
	BaseURL    string // default: https://api.webtor.io/v1
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name       store.StoreName
	baseURL    string
	apiKey     string
	httpClient *http.Client
	userAgent  string
}

var _ store.Store = (*StoreClient)(nil)

func NewStoreClient(conf *StoreClientConfig) *StoreClient {
	if conf == nil {
		conf = &StoreClientConfig{}
	}
	c := &StoreClient{}
	c.Name = store.StoreNameWebtor
	c.baseURL = conf.BaseURL
	c.apiKey = conf.APIKey
	c.httpClient = conf.HTTPClient
	if c.httpClient == nil {
		c.httpClient = config.DefaultHTTPClient
	}
	c.userAgent = conf.UserAgent
	if c.userAgent == "" {
		c.userAgent = "stremthru"
	}

	return c
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

// sdk builds an SDK client bound to the per-request API key.
func (c *StoreClient) sdk(apiKey string) (*webtorsdk.Client, error) {
	opts := []webtorsdk.WebUIOption{}
	if c.baseURL != "" {
		opts = append(opts, webtorsdk.WithWebUIBaseURL(c.baseURL))
	}
	backend, err := webtorsdk.WebUI(apiKey, opts...)
	if err != nil {
		serr := core.NewStoreError("failed to create client")
		serr.StoreName = string(c.Name)
		serr.Cause = err
		return nil, serr
	}
	client, err := webtorsdk.New(backend,
		webtorsdk.WithHTTPClient(c.httpClient),
		webtorsdk.WithUserAgent(c.userAgent),
	)
	if err != nil {
		serr := core.NewStoreError("failed to create client")
		serr.StoreName = string(c.Name)
		serr.Cause = err
		return nil, serr
	}
	return client, nil
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	start := time.Now()
	res, err := client.Profile(params.GetContext())
	stats.Record(c.Name, "get_user", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}
	// the API itself answers `payment_required` for free accounts, so a
	// successful call implies an active paid plan.
	user := &store.User{
		Id:                 res.UserID,
		Email:              res.Email,
		SubscriptionStatus: store.UserSubscriptionStatusPremium,
	}
	return user, nil
}

// listAllFiles pages through the resource's flat file listing and returns the
// files as store.MagnetFile with locked links.
func (c *StoreClient) listAllFiles(ctx context.Context, client *webtorsdk.Client, hash string) ([]store.MagnetFile, error) {
	files := []store.MagnetFile{}
	source := string(c.GetName().Code())
	start := time.Now()
	for item, err := range client.ListAll(ctx, hash, webtorsdk.ListOptions{}) {
		if err != nil {
			stats.Record(c.Name, "list_files", time.Since(start), true)
			return nil, UpstreamErrorWithCause(err)
		}
		if item.Type != webtorsdk.ListTypeFile {
			continue
		}
		files = append(files, store.MagnetFile{
			Idx:    item.Index,
			Link:   LockedFileLink("").Create(hash, item.Index),
			Name:   item.Name,
			Path:   item.Path,
			Size:   item.Size,
			Source: source,
		})
	}
	stats.Record(c.Name, "list_files", time.Since(start), false)
	return files, nil
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	magnetByHash := map[string]core.MagnetLink{}
	hashes := make([]string, len(params.Magnets))
	for i, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		magnetByHash[magnet.Hash] = magnet
		hashes[i] = magnet.Hash
	}

	foundItemByHash := map[string]store.CheckMagnetDataItem{}

	if data, err := buddy.CheckMagnet(c, hashes, params.GetAPIKey(c.apiKey), params.ClientIP, params.SId, false); err != nil {
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

	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	ctx := params.GetContext()

	type checkResult struct {
		res   *webtorsdk.ResourceResponse
		files []store.MagnetFile
	}
	resultByIdx := make([]*checkResult, len(missingHashes))
	g := errgroup.Group{}
	g.SetLimit(checkMagnetConcurrency)
	for i := range missingHashes {
		g.Go(func() error {
			hash := missingHashes[i]
			start := time.Now()
			res, err := client.Resource(ctx, hash)
			stats.Record(c.Name, "check_torz", time.Since(start), err != nil)
			if err != nil {
				if webtorsdk.IsNotFound(err) {
					return nil
				}
				return UpstreamErrorWithCause(err)
			}
			result := &checkResult{res: res}
			if files, err := c.listAllFiles(ctx, client, hash); err == nil {
				result.files = files
			}
			resultByIdx[i] = result
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	tByHash := map[string]*checkResult{}
	for i, hash := range missingHashes {
		if resultByIdx[i] != nil {
			tByHash[hash] = resultByIdx[i]
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
			tInfo.TorrentTitle = t.res.Name
			tInfo.Size = t.res.Size
			item.Status = store.MagnetStatusCached
			item.Name = t.res.Name
			item.Size = t.res.Size
			for i := range t.files {
				f := &t.files[i]
				tInfo.Files = append(tInfo.Files, torrent_stream.File{
					Idx:    f.Idx,
					Path:   f.Path,
					Name:   f.Name,
					Size:   f.Size,
					Source: f.Source,
				})
				item.Files = append(item.Files, *f)
			}
		}
		tInfos = append(tInfos, tInfo)
		data.Items = append(data.Items, item)
	}
	go buddy.BulkTrackMagnet(c, tInfos, nil, "", params.GetAPIKey(c.apiKey))
	return data, nil
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/webtor/"

func (l LockedFileLink) encodeData(hash string, idx int) string {
	return util.Base64Encode(hash + ":" + strconv.Itoa(idx))
}

func (l LockedFileLink) decodeData(encoded string) (hash string, idx int, err error) {
	decoded, err := util.Base64Decode(encoded)
	if err != nil {
		return "", 0, err
	}
	hash, idxStr, found := strings.Cut(decoded, ":")
	if !found {
		return "", 0, core.NewStoreError("invalid link")
	}
	idx, err = strconv.Atoi(idxStr)
	if err != nil {
		return "", 0, err
	}
	return hash, idx, nil
}

func (l LockedFileLink) Create(hash string, idx int) string {
	return lockedFileLinkPrefix + l.encodeData(hash, idx)
}

func (l LockedFileLink) Parse() (hash string, idx int, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	ctx := params.GetContext()

	var src webtorsdk.ResourceSource
	var magnet *core.MagnetLink

	if params.Magnet != "" {
		m, err := core.ParseMagnetLink(params.Magnet)
		if err != nil {
			return nil, err
		}
		magnet = &m
		src = webtorsdk.Magnet(m.RawLink)
	} else if params.Torrent != nil {
		f, err := params.Torrent.Open()
		if err != nil {
			return nil, err
		}
		defer f.Close()
		src = webtorsdk.TorrentReader(f)
	} else {
		err := core.NewStoreError("missing magnet or torrent")
		err.StoreName = string(c.Name)
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}

	start := time.Now()
	res, err := client.AddResource(ctx, src)
	stats.Record(c.Name, "add_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}
	if magnet == nil {
		m, err := core.ParseMagnetLink(res.ID)
		if err != nil {
			return nil, err
		}
		magnet = &m
	}

	start = time.Now()
	lib, err := client.LibraryAdd(ctx, res.ID)
	stats.Record(c.Name, "add_library", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	data := &store.AddMagnetData{
		Id:      res.ID,
		Hash:    res.ID,
		Magnet:  magnet.Link,
		Name:    res.Name,
		Size:    res.Size,
		Status:  store.MagnetStatusDownloaded,
		Files:   []store.MagnetFile{},
		AddedAt: lib.AddedAt,
	}

	files, err := c.listAllFiles(ctx, client, res.ID)
	if err != nil {
		return nil, err
	}
	data.Files = files

	return data, nil
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	ctx := params.GetContext()

	id := strings.ToLower(params.Id)
	start := time.Now()
	res, err := client.Resource(ctx, id)
	stats.Record(c.Name, "get_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	data := &store.GetMagnetData{
		Id:      res.ID,
		Hash:    res.ID,
		Name:    res.Name,
		Size:    res.Size,
		Status:  store.MagnetStatusDownloaded,
		Files:   []store.MagnetFile{},
		AddedAt: time.Now(),
	}

	if lib, err := client.LibraryGet(ctx, id); err == nil {
		data.AddedAt = lib.AddedAt
	}

	files, err := c.listAllFiles(ctx, client, res.ID)
	if err != nil {
		return nil, err
	}
	data.Files = files

	return data, nil
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	start := time.Now()
	res, err := client.LibraryList(params.GetContext(), webtorsdk.LibraryListOptions{
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	stats.Record(c.Name, "list_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}

	data := &store.ListMagnetsData{
		Items:      []store.ListMagnetsDataItem{},
		TotalItems: res.Count,
	}
	for i := range res.Items {
		item := &res.Items[i]
		data.Items = append(data.Items, store.ListMagnetsDataItem{
			Id:      item.ResourceID,
			Hash:    item.ResourceID,
			Name:    item.Name,
			Size:    item.Size,
			Status:  store.MagnetStatusDownloaded,
			AddedAt: item.AddedAt,
		})
	}
	return data, nil
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}
	id := strings.ToLower(params.Id)
	start := time.Now()
	err = client.LibraryRemove(params.GetContext(), id)
	stats.Record(c.Name, "remove_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}
	return &store.RemoveMagnetData{Id: id}, nil
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	hash, idx, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		aerr := core.NewAPIError("invalid link")
		aerr.StatusCode = http.StatusBadRequest
		aerr.Cause = err
		return nil, aerr
	}

	client, err := c.sdk(params.GetAPIKey(c.apiKey))
	if err != nil {
		return nil, err
	}

	start := time.Now()
	res, err := client.Export(params.GetContext(), hash, strconv.Itoa(idx), webtorsdk.ExportOptions{
		Types: []webtorsdk.ExportType{webtorsdk.ExportTypeDownload},
	})
	stats.Record(c.Name, "generate_torz_link", time.Since(start), err != nil)
	if err != nil {
		return nil, UpstreamErrorWithCause(err)
	}
	link, ok := res.DownloadURL()
	if !ok {
		serr := core.NewStoreError("missing download export")
		serr.StoreName = string(c.Name)
		serr.StatusCode = http.StatusNotFound
		return nil, serr
	}
	return &store.GenerateLinkData{Link: link}, nil
}
