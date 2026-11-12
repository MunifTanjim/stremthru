package highway

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/stats"
)

// HighWay keeps torrents in a shared cache and exposes them through its transfer
// API (https://cloud.high-way.me/center-api/transfers.php, bearer token).
//
// Torrents that are not cached yet wait in a queue on HighWay's side and get a
// temporary 32-hex id. Once they start they get a numeric id, so the store id of
// a queued torrent carries the info hash as well ("<queue id>-<hash>") to find it
// again.

const checkBatchSize = 100

var (
	queueIdPattern  = regexp.MustCompile(`^[a-f0-9]{32}$`)
	numericIdRegexp = regexp.MustCompile(`^[1-9][0-9]*$`)
	infoHashPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
)

type StoreClientConfig struct {
	BaseURL    string // default: https://cloud.high-way.me/center-api
	HTTPClient *http.Client
	UserAgent  string
}

type StoreClient struct {
	Name   store.StoreName
	client *APIClient
}

var _ store.Store = (*StoreClient)(nil)

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	if config == nil {
		config = &StoreClientConfig{}
	}
	c := &StoreClient{}
	c.client = NewAPIClient(&APIClientConfig{
		BaseURL:    config.BaseURL,
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
	})
	c.Name = store.StoreNameHighWay
	return c
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

func toMagnetStatus(state string) store.MagnetStatus {
	switch state {
	case "queued":
		return store.MagnetStatusQueued
	case "downloading":
		return store.MagnetStatusDownloading
	case "completed":
		return store.MagnetStatusDownloaded
	case "error":
		return store.MagnetStatusFailed
	default:
		return store.MagnetStatusUnknown
	}
}

// ------------------------------------------------------------------------------------------ ids

type transferId struct {
	id      string // numeric or 32-hex queue id
	hash    string // info hash (torrent) or NZB md5 (usenet), only for queue ids
	service string // "torrent" (default) or "usenet"
}

func (t transferId) serviceName() string {
	if t.service == "" {
		return "torrent"
	}
	return t.service
}

func (t transferId) isQueued() bool {
	return queueIdPattern.MatchString(t.id)
}

func (t transferId) query() url.Values {
	query := url.Values{"id": {t.id}}
	if !t.isQueued() {
		query.Set("service", t.serviceName())
	}
	return query
}

func storeId(id string, hash string) string {
	if queueIdPattern.MatchString(id) && hash != "" {
		return id + "-" + hash
	}
	return id
}

func parseStoreId(id string) (transferId, error) {
	if numericIdRegexp.MatchString(id) || queueIdPattern.MatchString(id) {
		return transferId{id: id}, nil
	}
	if queueId, hash, ok := strings.Cut(id, "-"); ok && queueIdPattern.MatchString(queueId) && infoHashPattern.MatchString(hash) {
		return transferId{id: queueId, hash: hash}, nil
	}
	err := core.NewAPIError("invalid id")
	err.StatusCode = http.StatusBadRequest
	return transferId{}, err
}

// ---------------------------------------------------------------------------------------- links

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/highway/"

func (l LockedFileLink) Create(transferId string, fileId string) string {
	return lockedFileLinkPrefix + util.Base64Encode(transferId+":"+fileId)
}

func (l LockedFileLink) Parse() (transferId string, fileId string, err error) {
	if !strings.HasPrefix(string(l), lockedFileLinkPrefix) {
		return "", "", errors.New("invalid link")
	}
	decoded, err := util.Base64Decode(strings.TrimPrefix(string(l), lockedFileLinkPrefix))
	if err != nil {
		return "", "", err
	}
	transferId, fileId, found := strings.Cut(decoded, ":")
	if !found || !numericIdRegexp.MatchString(transferId) || fileId == "" {
		return "", "", errors.New("invalid link")
	}
	return transferId, fileId, nil
}

func toMagnetFiles(t *transfer) []store.MagnetFile {
	files := []store.MagnetFile{}
	if t.State != "completed" {
		return files
	}
	source := string(store.StoreCodeHighWay)
	for _, f := range t.Files {
		if f.Partial {
			continue
		}
		idx := -1
		if f.Index != nil {
			idx = *f.Index
		}
		path := "/" + strings.TrimPrefix(f.Path, "/")
		if strings.Contains(strings.Trim(f.Path, "/"), "/") {
			path, _ = util.RemoveRootFolderFromPath(f.Path)
		}
		files = append(files, store.MagnetFile{
			Idx:    idx,
			Link:   LockedFileLink("").Create(t.Id, f.Id),
			Path:   path,
			Name:   f.Path[strings.LastIndex(f.Path, "/")+1:],
			Size:   f.Size,
			Source: source,
		})
	}
	return files
}

func toTime(unix *int64) time.Time {
	if unix == nil || *unix <= 0 {
		return time.Time{}
	}
	return time.Unix(*unix, 0).UTC()
}

func toHash(hash *string) string {
	if hash == nil {
		return ""
	}
	return strings.ToLower(*hash)
}

// ----------------------------------------------------------------------------------------- user

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	ctx := params.Ctx
	ctx.Query = &url.Values{"stats": {"1"}}
	response := &statsResponse{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, &ctx, response)
	stats.Record(s.Name, "get_user", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	user := &store.User{
		Id:                 strconv.FormatInt(response.UserId, 10),
		SubscriptionStatus: store.UserSubscriptionStatusExpired,
	}
	if response.Premium {
		user.SubscriptionStatus = store.UserSubscriptionStatusPremium
		// every HighWay plan includes Usenet
		user.HasUsenet = true
	}
	return user, nil
}

// ---------------------------------------------------------------------------------------- check

func (s *StoreClient) checkHashes(ctx store.Ctx, service string, hashes []string) (map[string]checkItem, error) {
	results := map[string]checkItem{}
	stat := "check_torz"
	if service == "usenet" {
		stat = "check_newz"
	}
	for i := 0; i < len(hashes); i += checkBatchSize {
		batch := hashes[i:min(i+checkBatchSize, len(hashes))]
		reqCtx := ctx
		reqCtx.Query = &url.Values{"service": {service}, "check": {strings.Join(batch, ",")}}
		response := &checkResponse{}
		start := time.Now()
		_, err := s.client.Request(http.MethodGet, &reqCtx, response)
		stats.Record(s.Name, stat, time.Since(start), err != nil)
		if err != nil {
			return nil, err
		}
		for hash, item := range response.Results {
			results[strings.ToLower(hash)] = item
		}
	}
	return results, nil
}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	magnetByHash := make(map[string]core.MagnetLink, len(params.Magnets))
	hashes := make([]string, 0, len(params.Magnets))
	for _, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		if _, seen := magnetByHash[magnet.Hash]; !seen {
			hashes = append(hashes, magnet.Hash)
		}
		magnetByHash[magnet.Hash] = magnet
	}

	foundItemByHash := map[string]store.CheckMagnetDataItem{}
	data, err := buddy.CheckMagnet(s, hashes, params.GetAPIKey(s.client.apiKey), params.ClientIP, params.SId, false)
	if err != nil {
		return nil, err
	}
	for _, item := range data.Items {
		foundItemByHash[item.Hash] = item
	}

	if params.LocalOnly {
		data := &store.CheckMagnetData{Items: []store.CheckMagnetDataItem{}}
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

	resultByHash := map[string]checkItem{}
	if len(missingHashes) > 0 {
		resultByHash, err = s.checkHashes(params.Ctx, "torrent", missingHashes)
		if err != nil {
			return nil, err
		}
	}

	result := &store.CheckMagnetData{Items: []store.CheckMagnetDataItem{}}
	tInfos := []buddy.TorrentInfoInput{}
	cached := map[string]bool{}
	for _, hash := range hashes {
		if item, ok := foundItemByHash[hash]; ok {
			result.Items = append(result.Items, item)
			continue
		}
		m := magnetByHash[hash]
		item := store.CheckMagnetDataItem{
			Hash:   m.Hash,
			Magnet: m.Link,
			Status: store.MagnetStatusUnknown,
			Files:  []store.MagnetFile{},
		}
		r := resultByHash[hash]
		cached[hash] = r.Ready
		if r.Ready {
			item.Status = store.MagnetStatusCached
			item.Size = r.Size
			if r.Name != nil {
				item.Name = *r.Name
			}
			tInfos = append(tInfos, buddy.TorrentInfoInput{
				Hash:         hash,
				TorrentTitle: item.Name,
				Size:         r.Size,
			})
		}
		result.Items = append(result.Items, item)
	}
	go buddy.BulkTrackMagnet(s, tInfos, cached, "", params.GetAPIKey(s.client.apiKey))
	return result, nil
}

// --------------------------------------------------------------------------------------- magnet

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	ctx := params.Ctx
	var magnet core.MagnetLink
	if params.Torrent != nil {
		mi, _, err := params.GetTorrentMeta()
		if err != nil {
			return nil, err
		}
		magnet, err = core.ParseMagnetLink(mi.HashInfoBytes().HexString())
		if err != nil {
			return nil, err
		}
		ctx.MultiPartForm = &multipart.Form{
			File: map[string][]*multipart.FileHeader{"file": {params.Torrent}},
		}
	} else {
		var err error
		magnet, err = core.ParseMagnetLink(params.Magnet)
		if err != nil {
			return nil, err
		}
		ctx.Form = &url.Values{"magnet": {magnet.RawLink}}
	}

	response := &addResponse{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, &ctx, response)
	stats.Record(s.Name, "add_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}

	data := &store.AddMagnetData{
		Id:      storeId(response.Id, magnet.Hash),
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    magnet.Name,
		Status:  toMagnetStatus(response.State),
		Files:   []store.MagnetFile{},
		AddedAt: time.Now().UTC(),
	}
	if !queueIdPattern.MatchString(response.Id) {
		// Known torrents are attached right away; fetch name, size and files.
		if t, err := s.fetchTransfer(params.Ctx, transferId{id: response.Id}); err == nil {
			data.Name = t.Name
			data.Size = t.Size
			data.Status = toMagnetStatus(t.State)
			data.Files = toMagnetFiles(t)
			if addedAt := toTime(t.AddedAt); !addedAt.IsZero() {
				data.AddedAt = addedAt
			}
		}
	}
	return data, nil
}

func isNotFound(err error) bool {
	var uerr *core.UpstreamError
	return errors.As(err, &uerr) && uerr.StatusCode == http.StatusNotFound
}

// fetchTransfer loads one transfer. A queued transfer that has started in the
// meantime is found again by its hash (info hash or NZB md5).
func (s *StoreClient) fetchTransfer(ctx store.Ctx, id transferId) (*transfer, error) {
	reqCtx := ctx
	query := id.query()
	reqCtx.Query = &query
	response := &transferResponse{}
	_, err := s.client.Request(http.MethodGet, &reqCtx, response)
	if err != nil && id.isQueued() && id.hash != "" && isNotFound(err) {
		results, checkErr := s.checkHashes(ctx, id.serviceName(), []string{id.hash})
		if checkErr != nil {
			return nil, checkErr
		}
		if r, ok := results[id.hash]; ok && r.Id != nil {
			return s.fetchTransfer(ctx, transferId{id: *r.Id, service: id.service})
		}
	}
	if err != nil {
		return nil, err
	}
	return &response.transfer, nil
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	id, err := parseStoreId(params.Id)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	t, err := s.fetchTransfer(params.Ctx, id)
	stats.Record(s.Name, "get_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	hash := toHash(t.Hash)
	if hash == "" {
		hash = id.hash
	}
	return &store.GetMagnetData{
		Id:      storeId(t.Id, hash),
		Name:    t.Name,
		Hash:    hash,
		Size:    t.Size,
		Status:  toMagnetStatus(t.State),
		Files:   toMagnetFiles(t),
		AddedAt: toTime(t.AddedAt),
	}, nil
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 500)
	offset := max(params.Offset, 0)

	ctx := params.Ctx
	ctx.Query = &url.Values{
		"service": {"torrent"},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(offset)},
	}
	response := &listResponse{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, &ctx, response)
	stats.Record(s.Name, "list_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}

	data := &store.ListMagnetsData{
		Items:      make([]store.ListMagnetsDataItem, 0, len(response.Transfers)),
		TotalItems: response.Total,
	}
	for i := range response.Transfers {
		t := &response.Transfers[i]
		hash := toHash(t.Hash)
		data.Items = append(data.Items, store.ListMagnetsDataItem{
			Id:      storeId(t.Id, hash),
			Hash:    hash,
			Name:    t.Name,
			Size:    t.Size,
			Status:  toMagnetStatus(t.State),
			AddedAt: toTime(t.AddedAt),
		})
	}
	return data, nil
}

func (s *StoreClient) remove(ctx store.Ctx, id transferId) (*removeResponse, error) {
	reqCtx := ctx
	query := id.query()
	reqCtx.Query = &query
	response := &removeResponse{}
	_, err := s.client.Request(http.MethodDelete, &reqCtx, response)
	return response, err
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	id, err := parseStoreId(params.Id)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	response, err := s.remove(params.Ctx, id)
	if err == nil && !response.Removed && response.Reason != "not_owner" && id.isQueued() && id.hash != "" {
		// The queued transfer may have started in the meantime.
		var results map[string]checkItem
		results, err = s.checkHashes(params.Ctx, "torrent", []string{id.hash})
		if r, ok := results[id.hash]; err == nil && ok && r.Id != nil {
			response, err = s.remove(params.Ctx, transferId{id: *r.Id})
		}
	}
	stats.Record(s.Name, "remove_torz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if !response.Removed {
		if response.Reason == "not_owner" {
			err := core.NewStoreError("torrent was added with another API token; a token with torrent:delete_self can only remove its own torrents")
			err.StoreName = string(s.Name)
			err.StatusCode = http.StatusForbidden
			err.Code = core.ErrorCodeForbidden
			return nil, err
		}
		err := core.NewStoreError("torrent not found")
		err.StoreName = string(s.Name)
		err.StatusCode = http.StatusNotFound
		err.Code = core.ErrorCodeNotFound
		return nil, err
	}
	return &store.RemoveMagnetData{Id: params.Id}, nil
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	tId, fileId, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	ctx := params.Ctx
	ctx.Query = &url.Values{"id": {tId}, "service": {"torrent"}, "file": {fileId}}
	response := &fileResponse{}
	start := time.Now()
	_, err = s.client.Request(http.MethodGet, &ctx, response)
	stats.Record(s.Name, "generate_torz_link", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if response.File.DownloadURL == nil || *response.File.DownloadURL == "" {
		err := core.NewStoreError("file is not ready yet")
		err.StoreName = string(s.Name)
		err.StatusCode = http.StatusConflict
		err.Code = core.ErrorCodeConflict
		return nil, err
	}
	return &store.GenerateLinkData{Link: *response.File.DownloadURL}, nil
}
