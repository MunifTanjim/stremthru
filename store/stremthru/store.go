package stremthru

import (
	"errors"
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

var (
	_ store.Store     = (*StoreClient)(nil)
	_ store.NewzStore = (*StoreClient)(nil)
)

type StoreClientConfig struct {
}

type StoreClient struct {
	Name store.StoreName
}

func NewStoreClient(config *StoreClientConfig) *StoreClient {
	c := &StoreClient{}
	c.Name = store.StoreNameStremThru
	return c
}

func notImplementedError() error {
	err := core.NewStoreError("not implemented")
	err.StoreName = string(store.StoreNameStremThru)
	err.StatusCode = http.StatusNotImplemented
	return err
}

func notFoundError() error {
	err := core.NewStoreError("not found")
	err.StoreName = string(store.StoreNameStremThru)
	err.StatusCode = http.StatusNotFound
	return err
}

func invalidAPIKeyError() error {
	err := core.NewStoreError("invalid api key")
	err.StoreName = string(store.StoreNameStremThru)
	err.StatusCode = http.StatusForbidden
	return err
}

func (c *StoreClient) ensureAuthed(params *store.Ctx) (*util.BasicAuth, error) {
	ba, err := util.ParseBasicAuth(params.APIKey)
	if err != nil || ba.Password == "" || ba.Password != config.UserAuth.GetPassword(ba.Username) {
		return nil, invalidAPIKeyError()
	}
	return &ba, nil
}

func (c *StoreClient) GetName() store.StoreName {
	return c.Name
}

func (c *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	ba, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	user := &store.User{
		Id:                 ba.Username,
		SubscriptionStatus: store.UserSubscriptionStatusPremium,
	}

	pool, err := usenetmanager.GetPool()
	if err == nil && pool != nil && pool.CountProviders() > 0 {
		user.HasUsenet = true
	}

	return user, nil
}

func (c *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	return nil, notImplementedError()
}

func (c *StoreClient) CheckNewz(params *store.CheckNewzParams) (*store.CheckNewzData, error) {
	_, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	items := make([]store.CheckNewzDataItem, len(params.Hashes))
	for i, hash := range params.Hashes {
		info, err := nzb_info.GetByHash(hash)
		if err != nil {
			return nil, err
		}
		item := store.CheckNewzDataItem{
			Hash:   hash,
			Status: store.NewzStatusUnknown,
		}
		if info != nil && info.Streamable {
			item.Status = store.NewzStatusCached
			item.Files = flattenContentFiles(info.ContentFiles.Data, "")
		}
		items[i] = item
	}
	return &store.CheckNewzData{Items: items}, nil
}

func (c *StoreClient) AddNewz(params *store.AddNewzParams) (*store.AddNewzData, error) {
	ba, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	if params.Link == "" {
		err := core.NewStoreError("missing link")
		err.StoreName = string(store.StoreNameStremThru)
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}

	id, err := nzb_info.QueueJob(ba.Username, "", params.Link, "", 0, "")
	if err != nil {
		return nil, err
	}

	data := &store.AddNewzData{
		Id:     id,
		Hash:   id,
		Status: store.NewzStatusQueued,
	}
	return data, nil
}

func flattenContentFiles(files []usenet_pool.NZBContentFile, parentPath string) []store.NewzFile {
	var result []store.NewzFile
	for _, f := range files {
		filePath := "/" + f.Name
		if parentPath != "" {
			filePath = parentPath + "::" + filePath
		}
		if len(f.Files) > 0 {
			for _, f := range f.Parts {
				if f.Size == 0 {
					continue
				}
				partPath := "/" + f.Name
				if parentPath != "" {
					partPath = parentPath + "::" + partPath
				}
				result = append(result, store.NewzFile{
					Idx:  -1,
					Path: partPath,
					Name: f.Name,
					Size: f.Size,
				})
			}
			result = append(result, flattenContentFiles(f.Files, filePath)...)
		} else {
			if f.Size == 0 {
				continue
			}
			result = append(result, store.NewzFile{
				Idx:  -1,
				Path: filePath,
				Name: f.Name,
				Size: f.Size,
			})
		}
	}
	return result
}

func mapJobStatusToNewzStatus(status string) store.NewzStatus {
	switch job_queue.EntryStatus(status) {
	case job_queue.EntryStatusQueued:
		return store.NewzStatusQueued
	case job_queue.EntryStatusProcessing:
		return store.NewzStatusProcessing
	case job_queue.EntryStatusFailed:
		return store.NewzStatusQueued
	case job_queue.EntryStatusDead:
		return store.NewzStatusFailed
	case job_queue.EntryStatusDone:
		return store.NewzStatusDownloaded
	default:
		return store.NewzStatusUnknown
	}
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/stremthru/"

func (l LockedFileLink) encodeData(id, path string) string {
	return util.Base64Encode(id + ":" + path)
}

func (l LockedFileLink) decodeData(encoded string) (id, path string, err error) {
	decoded, err := util.Base64Decode(encoded)
	if err != nil {
		return "", "", err
	}
	id, path, found := strings.Cut(decoded, ":")
	if !found {
		return "", "", errors.New("invalid locked file link")
	}
	return id, path, nil
}

func (l LockedFileLink) Create(id, path string) string {
	return lockedFileLinkPrefix + l.encodeData(id, path)
}

func (l LockedFileLink) Parse() (id, path string, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (c *StoreClient) GetNewz(params *store.GetNewzParams) (*store.GetNewzData, error) {
	_, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	info, err := nzb_info.GetByHash(params.Id)
	if err != nil {
		return nil, err
	}

	if info != nil {
		data := &store.GetNewzData{
			Id:      info.Hash,
			Hash:    info.Hash,
			Name:    info.Name,
			Size:    info.Size,
			Status:  store.NewzStatusUnknown,
			AddedAt: info.UAt.Time,
		}
		if info.Streamable {
			data.Status = store.NewzStatusDownloaded
			files := flattenContentFiles(info.ContentFiles.Data, "")
			for i := range files {
				file := &files[i]
				file.Link = LockedFileLink("").Create(info.Hash, file.Path)
			}
			data.Files = files
		}
		return data, nil
	}

	job, err := nzb_info.GetJobById(params.Id)
	if err != nil {
		return nil, err
	}
	if job != nil {
		data := &store.GetNewzData{
			Id:     job.Key,
			Hash:   job.Key,
			Name:   job.Payload.Data.Name,
			Status: mapJobStatusToNewzStatus(job.Status),
			Files:  []store.NewzFile{},
		}
		return data, nil
	}

	return nil, notFoundError()
}

func (c *StoreClient) ListNewz(params *store.ListNewzParams) (*store.ListNewzData, error) {
	_, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	allInfos, err := nzb_info.GetAll()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(allInfos))
	items := []store.ListNewzDataItem{}
	for _, info := range allInfos {
		seen[info.Hash] = struct{}{}
		if !info.Streamable {
			continue
		}
		items = append(items, store.ListNewzDataItem{
			Id:      info.Hash,
			Hash:    info.Hash,
			Name:    info.Name,
			Size:    info.Size,
			Status:  store.NewzStatusDownloaded,
			AddedAt: info.CAt.Time,
		})
	}

	jobs, err := nzb_info.GetAllJob()
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if _, exists := seen[job.Key]; exists {
			continue
		}
		items = append(items, store.ListNewzDataItem{
			Id:      job.Key,
			Hash:    job.Key,
			Name:    job.Payload.Data.Name,
			Status:  mapJobStatusToNewzStatus(job.Status),
			AddedAt: job.CreatedAt.Time,
		})
	}

	total := len(items)
	items = items[min(offset, total):min(offset+limit, total)]

	return &store.ListNewzData{
		Items:      items,
		TotalItems: total,
	}, nil
}

func (c *StoreClient) RemoveNewz(params *store.RemoveNewzParams) (*store.RemoveNewzData, error) {
	_, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	info, err := nzb_info.GetByHash(params.Id)
	if err != nil {
		return nil, err
	}

	if info != nil {
		if err := nzb_info.DeleteById(info.Id); err != nil {
			return nil, err
		}
	}

	err = nzb_info.DeleteJob(params.Id)
	if err != nil {
		return nil, err
	}

	return &store.RemoveNewzData{Id: params.Id}, nil
}

type newzStreamTokenData struct {
	EncLink   string `json:"enc_link"`
	EncFormat string `json:"enc_format"`
}

func (c *StoreClient) GenerateNewzLink(params *store.GenerateNewzLinkParams) (*store.GenerateNewzLinkData, error) {
	ba, err := c.ensureAuthed(&params.Ctx)
	if err != nil {
		return nil, err
	}
	id, path, err := LockedFileLink(params.Link).Parse()
	if err != nil {
		return nil, err
	}

	newz, err := c.GetNewz(&store.GetNewzParams{
		Ctx:      params.Ctx,
		Id:       id,
		ClientIP: params.ClientIP,
	})
	if err != nil {
		return nil, err
	}

	var file *store.NewzFile
	for i := range newz.Files {
		f := &newz.Files[i]
		if f.Path == path {
			file = f
			break
		}
	}

	if file == nil {
		return nil, notFoundError()
	}

	encLink, err := core.Encrypt(ba.Password, params.Link)
	if err != nil {
		return nil, err
	}

	claims := core.JWTClaims[newzStreamTokenData]{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "stremthru",
			Subject:   ba.Username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
		},
		Data: &newzStreamTokenData{
			EncLink:   encLink,
			EncFormat: core.EncryptionFormat,
		},
	}
	token, err := core.CreateJWT(ba.Password, claims)
	if err != nil {
		return nil, err
	}

	link := config.BaseURL.JoinPath("/v0/store/newz/stream", token, file.Name).String()

	return &store.GenerateNewzLinkData{Link: link}, nil
}

type unwrappedNewzStreamTokenData struct {
	User     string `json:"u"`
	ID       string `json:"id"`
	FilePath string `json:"fp"`
}

var newzStreamTokenDataCache = cache.NewCache[unwrappedNewzStreamTokenData](&cache.CacheConfig{
	Name:     "store:stremthru:newz-stream-token",
	Lifetime: 30 * time.Minute,
})

func getUserCredsFromJWT(t *jwt.Token) (user, password string, err error) {
	user, err = t.Claims.GetSubject()
	if err != nil {
		return "", "", err
	}
	password = config.UserAuth.GetPassword(user)
	return user, password, nil
}

func UnwrapNewzStreamToken(encodedToken string) (user, id, path string, err error) {
	linkData := &unwrappedNewzStreamTokenData{}
	if found := newzStreamTokenDataCache.Get(encodedToken, linkData); found {
		return linkData.User, linkData.ID, linkData.FilePath, nil
	}

	claims := &core.JWTClaims[newzStreamTokenData]{}
	password := ""
	_, err = core.ParseJWT(func(t *jwt.Token) (any, error) {
		user, password, err = getUserCredsFromJWT(t)
		return []byte(password), err
	}, encodedToken, claims)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenInvalidClaims) {
			rerr := core.NewAPIError("unauthorized")
			rerr.StatusCode = http.StatusUnauthorized
			rerr.Cause = err
			err = rerr
		}

		return "", "", "", err
	}

	link, err := core.Decrypt(password, claims.Data.EncLink)
	if err != nil {
		return "", "", "", err
	}

	id, path, err = LockedFileLink(link).Parse()
	if err != nil {
		return "", "", "", err
	}

	linkData.User = user
	linkData.ID = id
	linkData.FilePath = path

	newzStreamTokenDataCache.Add(encodedToken, *linkData)

	return linkData.User, linkData.ID, linkData.FilePath, nil
}
