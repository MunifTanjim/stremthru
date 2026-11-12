package highway

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/stats"
)

// Usenet on HighWay: NZBs are added by link (nzb_url, HighWay fetches the file)
// or as a file. HighWay knows an NZB by the md5 of its content, and also by the
// link hash (util.HashNZBFileLink) of every link it was added with, so CheckNewz
// can ask with the hashes StremThru uses.
//
// Like torrents, an NZB that waits in HighWay's queue has a temporary 32-hex id;
// its store id carries the md5 of the NZB content ("<queue id>-<md5>").

var _ store.NewzStore = (*StoreClient)(nil)

var nzbHashPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func toNewzStatus(state string) store.NewzStatus {
	switch state {
	case "queued":
		return store.NewzStatusQueued
	case "downloading":
		return store.NewzStatusDownloading
	case "completed":
		return store.NewzStatusDownloaded
	case "error":
		return store.NewzStatusFailed
	default:
		return store.NewzStatusUnknown
	}
}

func newzStoreId(id string, hash string) string {
	if queueIdPattern.MatchString(id) && hash != "" {
		return id + "-" + hash
	}
	return id
}

func parseNewzStoreId(id string) (transferId, error) {
	if numericIdRegexp.MatchString(id) || queueIdPattern.MatchString(id) {
		return transferId{id: id, service: "usenet"}, nil
	}
	if queueId, hash, ok := strings.Cut(id, "-"); ok && queueIdPattern.MatchString(queueId) && nzbHashPattern.MatchString(hash) {
		return transferId{id: queueId, hash: hash, service: "usenet"}, nil
	}
	err := core.NewAPIError("invalid id")
	err.StatusCode = http.StatusBadRequest
	return transferId{}, err
}

func toNewzFiles(t *transfer) []store.NewzFile {
	files := []store.NewzFile{}
	if t.State != "completed" {
		return files
	}
	for i, f := range t.Files {
		if f.Partial {
			continue
		}
		path := "/" + strings.TrimPrefix(f.Path, "/")
		if strings.Contains(strings.Trim(f.Path, "/"), "/") {
			path, _ = util.RemoveRootFolderFromPath(f.Path)
		}
		files = append(files, store.NewzFile{
			Idx:  i,
			Link: LockedFileLink("").Create(t.Id, f.Id),
			Path: path,
			Name: f.Path[strings.LastIndex(f.Path, "/")+1:],
			Size: f.Size,
		})
	}
	return files
}

func fileMD5(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *StoreClient) CheckNewz(params *store.CheckNewzParams) (*store.CheckNewzData, error) {
	hashes := make([]string, 0, len(params.Hashes))
	seen := map[string]bool{}
	for _, hash := range params.Hashes {
		hash = strings.ToLower(hash)
		if !seen[hash] && nzbHashPattern.MatchString(hash) {
			seen[hash] = true
			hashes = append(hashes, hash)
		}
	}
	results := map[string]checkItem{}
	if len(hashes) > 0 {
		var err error
		results, err = s.checkHashes(params.Ctx, "usenet", hashes)
		if err != nil {
			return nil, err
		}
	}
	data := &store.CheckNewzData{Items: []store.CheckNewzDataItem{}}
	for _, hash := range params.Hashes {
		item := store.CheckNewzDataItem{
			Hash:   hash,
			Status: store.NewzStatusUnknown,
			Files:  []store.NewzFile{},
		}
		if r, ok := results[strings.ToLower(hash)]; ok && r.Ready {
			item.Status = store.NewzStatusCached
		}
		data.Items = append(data.Items, item)
	}
	return data, nil
}

func (s *StoreClient) AddNewz(params *store.AddNewzParams) (*store.AddNewzData, error) {
	ctx := params.Ctx
	contentHash := ""
	if params.File != nil {
		var err error
		if contentHash, err = fileMD5(params.File); err != nil {
			return nil, err
		}
		ctx.MultiPartForm = &multipart.Form{
			File: map[string][]*multipart.FileHeader{"file": {params.File}},
		}
	} else if params.Link != "" {
		ctx.Form = &url.Values{"nzb_url": {params.Link}}
	} else {
		err := core.NewAPIError("missing nzb link or file")
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}

	response := &addResponse{}
	start := time.Now()
	_, err := s.client.Request(http.MethodPost, &ctx, response)
	stats.Record(s.Name, "add_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if response.Hash != "" {
		contentHash = strings.ToLower(response.Hash)
	}
	hash := contentHash
	if response.LinkHash != "" {
		// the hash StremThru uses for links, so the result matches CheckNewz
		hash = strings.ToLower(response.LinkHash)
	}
	return &store.AddNewzData{
		Id:     newzStoreId(response.Id, contentHash),
		Hash:   hash,
		Status: toNewzStatus(response.State),
	}, nil
}

func (s *StoreClient) GetNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
	id, err := parseNewzStoreId(params.Id)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	t, err := s.fetchTransfer(params.Ctx, id)
	stats.Record(s.Name, "get_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	hash := toHash(t.Hash)
	if hash == "" {
		hash = id.hash
	}
	return &store.GetNewzData{
		Id:      newzStoreId(t.Id, hash),
		Hash:    hash,
		Name:    t.Name,
		Size:    t.Size,
		Status:  toNewzStatus(t.State),
		Files:   toNewzFiles(t),
		AddedAt: toTime(t.AddedAt),
	}, nil
}

func (s *StoreClient) ListNewz(params *store.ListNewzParams) (*store.ListNewzData, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 500)
	offset := max(params.Offset, 0)

	ctx := params.Ctx
	ctx.Query = &url.Values{
		"service": {"usenet"},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(offset)},
	}
	response := &listResponse{}
	start := time.Now()
	_, err := s.client.Request(http.MethodGet, &ctx, response)
	stats.Record(s.Name, "list_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	data := &store.ListNewzData{
		Items:      make([]store.ListNewzDataItem, 0, len(response.Transfers)),
		TotalItems: response.Total,
	}
	for i := range response.Transfers {
		t := &response.Transfers[i]
		hash := toHash(t.Hash)
		data.Items = append(data.Items, store.ListNewzDataItem{
			Id:      newzStoreId(t.Id, hash),
			Hash:    hash,
			Name:    t.Name,
			Size:    t.Size,
			Status:  toNewzStatus(t.State),
			AddedAt: toTime(t.AddedAt),
		})
	}
	return data, nil
}

func (s *StoreClient) RemoveNewz(params *store.RemoveNewzParams) (*store.RemoveNewzData, error) {
	id, err := parseNewzStoreId(params.Id)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	response, err := s.remove(params.Ctx, id)
	if err == nil && !response.Removed && response.Reason != "not_owner" && id.isQueued() && id.hash != "" {
		// The queued NZB may have started in the meantime.
		var results map[string]checkItem
		results, err = s.checkHashes(params.Ctx, "usenet", []string{id.hash})
		if r, ok := results[id.hash]; err == nil && ok && r.Id != nil {
			response, err = s.remove(params.Ctx, transferId{id: *r.Id, service: "usenet"})
		}
	}
	stats.Record(s.Name, "remove_newz", time.Since(start), err != nil)
	if err != nil {
		return nil, err
	}
	if !response.Removed {
		if response.Reason == "not_owner" {
			err := core.NewStoreError("nzb was added with another API token; a token with usenet:delete_self can only remove its own downloads")
			err.StoreName = string(s.Name)
			err.StatusCode = http.StatusForbidden
			err.Code = core.ErrorCodeForbidden
			return nil, err
		}
		err := core.NewStoreError("nzb not found")
		err.StoreName = string(s.Name)
		err.StatusCode = http.StatusNotFound
		err.Code = core.ErrorCodeNotFound
		return nil, err
	}
	return &store.RemoveNewzData{Id: params.Id}, nil
}

func (s *StoreClient) GenerateNewzLink(params *store.GenerateNewzLinkParams) (*store.GenerateNewzLinkData, error) {
	tId, fileId, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	ctx := params.Ctx
	ctx.Query = &url.Values{"id": {tId}, "service": {"usenet"}, "file": {fileId}}
	response := &fileResponse{}
	start := time.Now()
	_, err = s.client.Request(http.MethodGet, &ctx, response)
	stats.Record(s.Name, "generate_newz_link", time.Since(start), err != nil)
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
	return &store.GenerateNewzLinkData{Link: *response.File.DownloadURL}, nil
}
