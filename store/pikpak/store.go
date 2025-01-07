package pikpak

import (
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/store"
)

func toSize(sizeStr string) int {
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		size = -1
	}
	return size
}

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
	c.Name = store.StoreNamePikPak

	return c
}

func (s *StoreClient) GetName() store.StoreName {
	return s.Name
}

func (s *StoreClient) getTask(ctx Ctx, taskId string) (*Task, error) {
	res, err := s.client.ListTasks(&ListTasksParams{
		Ctx:   ctx,
		Limit: 200,
		Filters: map[string]map[string]any{
			"phase": map[string]any{
				"in": FilePhaseRunning + "," + FilePhaseError + "," + FilePhaseComplete,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	for i := range res.Data.Tasks {
		t := &res.Data.Tasks[i]
		if t.Id == taskId {
			return t, nil
		}
	}
	error := core.NewStoreError("task not found: " + string(taskId))
	error.StoreName = string(s.GetName())
	return nil, error
}

func (s *StoreClient) waitForTaskComplete(ctx Ctx, taskId string, maxRetry int, retryInterval time.Duration) (*Task, error) {
	t, err := s.getTask(ctx, taskId)
	if err != nil {
		return nil, err
	}
	retry := 0
	for (t.Phase != FilePhaseComplete && t.Phase != FilePhaseError) && retry < maxRetry {
		time.Sleep(retryInterval)
		task, err := s.getTask(ctx, t.Id)
		if err != nil {
			return t, err
		}
		t = task
		retry++
	}
	if t.Phase != FilePhaseComplete {
		error := core.NewStoreError("task failed to reach phase: " + string(FilePhaseComplete))
		error.StoreName = string(s.GetName())
		return t, error
	}
	return t, nil
}

func (s *StoreClient) AddMagnet(params *store.AddMagnetParams) (*store.AddMagnetData, error) {
	magnet, err := core.ParseMagnetLink(params.Magnet)
	if err != nil {
		return nil, err
	}
	ctx := Ctx{Ctx: params.Ctx}
	res, err := s.client.AddFile(&AddFileParams{
		Ctx: ctx,
		URL: AddFileParamsURL{
			URL: magnet.Link,
		},
	})
	if err != nil {
		return nil, err
	}
	data := &store.AddMagnetData{
		Id:      res.Data.Task.FileId,
		Hash:    magnet.Hash,
		Magnet:  magnet.Link,
		Name:    "",
		Status:  store.MagnetStatusDownloading,
		Files:   []store.MagnetFile{},
		AddedAt: time.Now(),
	}
	if task, err := s.waitForTaskComplete(ctx, res.Data.Task.Id, 3, 5*time.Second); task != nil {
		if err != nil {
			log.Printf("[pikpak] error waiting for task complete %v\n", err)
		}
		if task != nil {
			if task.Phase == FilePhaseComplete {
				mRes, err := s.GetMagnet(&store.GetMagnetParams{
					Ctx: ctx.Ctx,
					Id:  data.Id,
				})
				if err != nil {
					return nil, err
				}
				data.Name = mRes.Name
				data.Status = mRes.Status
				data.Files = mRes.Files
				data.AddedAt = mRes.AddedAt
			} else if task.Phase == FilePhaseError {
				data.Status = store.MagnetStatusFailed
			}
		}
	}
	return data, nil
}

func (s *StoreClient) CheckMagnet(params *store.CheckMagnetParams) (*store.CheckMagnetData, error) {
	hashes := []string{}
	for _, m := range params.Magnets {
		magnet, err := core.ParseMagnetLink(m)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, magnet.Hash)
	}

	data, err := buddy.CheckMagnet(s, hashes, params.GetAPIKey(s.client.apiKey), params.ClientIP, params.SId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type LockedFileLink string

const lockedFileLinkPrefix = "stremthru://store/pikpak/"

func (l LockedFileLink) encodeData(magnetId, fileId string) string {
	return core.Base64Encode(magnetId + ":" + fileId)
}

func (l LockedFileLink) decodeData(encoded string) (magnetId, fileId string, err error) {
	decoded, err := core.Base64Decode(encoded)
	if err != nil {
		return "", "", err
	}
	magnetId, fileId, found := strings.Cut(decoded, ":")
	if !found {
		return "", "", err
	}
	return magnetId, fileId, nil
}

func (l LockedFileLink) create(magnetId, fileId string) string {
	return lockedFileLinkPrefix + l.encodeData(magnetId, fileId)
}

func (l LockedFileLink) parse() (magnetId, fileId string, err error) {
	encoded := strings.TrimPrefix(string(l), lockedFileLinkPrefix)
	return l.decodeData(encoded)
}

func (s *StoreClient) GenerateLink(params *store.GenerateLinkParams) (*store.GenerateLinkData, error) {
	_, fileId, err := LockedFileLink(params.Link).parse()
	if err != nil {
		error := core.NewAPIError("invalid link")
		error.StoreName = string(s.GetName())
		error.StatusCode = http.StatusBadRequest
		error.Cause = err
		return nil, error
	}
	ctx := Ctx{Ctx: params.Ctx}
	res, err := s.client.GetFile(&GetFileParams{
		Ctx:    ctx,
		FileId: fileId,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data.Medias) == 0 {
		err := core.NewStoreError("file not found")
		err.StoreName = string(s.GetName())
		err.StatusCode = http.StatusNotFound
		return nil, err
	}
	data := &store.GenerateLinkData{
		Link: res.Data.Medias[0].Link.URL,
	}
	return data, nil
}

func (c *StoreClient) listFilesFlat(ctx Ctx, folderId string, result []store.MagnetFile, parent *store.MagnetFile, idx int, rootFolderId string) ([]store.MagnetFile, error) {
	if result == nil {
		result = []store.MagnetFile{}
	}

	params := &ListFilesParams{
		Ctx:      ctx,
		ParentId: folderId,
		Filters: map[string]map[string]any{
			"trashed": map[string]any{"eq": false},
			"phase":   map[string]any{"eq": FilePhaseComplete},
		},
	}
	lfRes, err := c.client.ListFiles(params)
	if err != nil {
		return nil, err
	}

	for _, f := range lfRes.Data.Files {
		file := &store.MagnetFile{
			Idx:  -1, // order is non-deterministic
			Link: LockedFileLink("").create(rootFolderId, f.Id),
			Name: f.Name,
			Path: "/" + f.Name,
			Size: toSize(f.Size),
		}

		if parent != nil {
			file.Path = path.Join(parent.Path, file.Name)
		}

		if f.Kind == FileKindFolder {
			result, err = c.listFilesFlat(ctx, f.Id, result, file, idx, rootFolderId)
			if err != nil {
				return nil, err
			}
			idx = len(result)
		} else {
			result = append(result, *file)
			idx++
		}
	}

	return result, nil
}

func (s *StoreClient) GetMagnet(params *store.GetMagnetParams) (*store.GetMagnetData, error) {
	ctx := Ctx{Ctx: params.Ctx}
	res, err := s.client.GetFile(&GetFileParams{
		Ctx:    ctx,
		FileId: params.Id,
	})
	if err != nil {
		return nil, err
	}
	magnet, err := core.ParseMagnetLink(res.Data.Params.URL)
	if err != nil {
		return nil, err
	}
	addedAt, err := time.Parse(time.RFC3339, res.Data.CreatedTime)
	if err != nil {
		addedAt = time.Now()
	}
	data := &store.GetMagnetData{
		Id:      res.Data.Id,
		Name:    res.Data.Name,
		Hash:    magnet.Hash,
		Status:  store.MagnetStatusDownloading,
		Files:   []store.MagnetFile{},
		AddedAt: addedAt,
	}
	if res.Data.Phase == FilePhaseComplete {
		data.Status = store.MagnetStatusDownloaded
		if res.Data.Kind == FileKindFolder {
			files, err := s.listFilesFlat(ctx, data.Id, nil, nil, 0, data.Id)
			if err != nil {
				return nil, err
			}
			data.Files = files
		} else {
			data.Files = append(data.Files, store.MagnetFile{
				Idx:  0,
				Link: LockedFileLink("").create(data.Id, ""),
				Name: data.Name,
				Path: "/" + data.Name,
				Size: toSize(res.Data.Size),
			})
		}
	}
	return data, nil
}

func (s *StoreClient) GetUser(params *store.GetUserParams) (*store.User, error) {
	res, err := s.client.GetUser(&GetUserParams{
		Ctx: Ctx{Ctx: params.Ctx},
	})
	if err != nil {
		return nil, err
	}
	vipRes, err := s.client.GetVIPInfo(&GetVIPInfoParams{
		Ctx: Ctx{Ctx: params.Ctx},
	})
	if err != nil {
		return nil, err
	}
	data := &store.User{
		Id:                 res.Data.Sub,
		Email:              res.Data.Email,
		SubscriptionStatus: store.UserSubscriptionStatusTrial,
	}
	if vipRes.Data.Type == VIPTypePlatinum {
		data.SubscriptionStatus = store.UserSubscriptionStatusPremium
	}
	return data, nil
}

func (s *StoreClient) getMyPackFolder(ctx Ctx) (*File, error) {
	res, err := s.client.ListFiles(&ListFilesParams{
		Ctx: ctx,
		Filters: map[string]map[string]any{
			"trashed": map[string]any{"eq": false},
			"phase":   map[string]any{"eq": FilePhaseComplete},
		},
	})
	if err != nil {
		return nil, err
	}
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		if f.Name == "My Pack" {
			return f, nil
		}
	}
	err = core.NewAPIError("'My Pack' folder missing")
	return nil, err
}

func (s *StoreClient) ListMagnets(params *store.ListMagnetsParams) (*store.ListMagnetsData, error) {
	ctx := Ctx{Ctx: params.Ctx}
	myPackFolder, err := s.getMyPackFolder(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.client.ListFiles(&ListFilesParams{
		Ctx:      Ctx{Ctx: params.Ctx},
		ParentId: myPackFolder.Id,
		Filters: map[string]map[string]any{
			"trashed": map[string]any{"eq": false},
			"phase":   map[string]any{"eq": FilePhaseComplete},
		},
	})
	if err != nil {
		return nil, err
	}
	data := &store.ListMagnetsData{
		Items: []store.ListMagnetsDataItem{},
	}
	for i := range res.Data.Files {
		f := &res.Data.Files[i]
		addedAt, err := time.Parse(time.RFC3339, f.CreatedTime)
		if err != nil {
			addedAt = time.Now()
		}
		if !strings.HasPrefix(f.Params.URL, "magnet:") {
			continue
		}
		magnet, err := core.ParseMagnetLink(f.Params.URL)
		if err != nil {
			continue
		}
		item := store.ListMagnetsDataItem{
			Id:      f.Id,
			Name:    f.Name,
			Hash:    magnet.Hash,
			Status:  store.MagnetStatusDownloading,
			AddedAt: addedAt,
		}
		if f.Phase == FilePhaseComplete {
			item.Status = store.MagnetStatusDownloaded
		}
		data.Items = append(data.Items, item)
	}
	data.TotalItems = len(data.Items)
	return data, nil
}

func (s *StoreClient) RemoveMagnet(params *store.RemoveMagnetParams) (*store.RemoveMagnetData, error) {
	_, err := s.client.Trash(&TrashParams{
		Ctx: Ctx{Ctx: params.Ctx},
		Ids: []string{params.Id},
	})
	if err != nil {
		return nil, err
	}
	data := &store.RemoveMagnetData{
		Id: params.Id,
	}
	return data, nil
}
