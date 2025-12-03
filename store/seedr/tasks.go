package seedr

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type AddTaskData struct {
	ResponseContainer
	UserTorrentId string `json:"user_torrent_id"`
	Title         string `json:"title"`
	Success       bool   `json:"success"`
	TorrentHash   string `json:"torrent_hash"`
}

type AddTaskParams struct {
	Ctx
	FolderId      int64
	TorrentMagnet string
	TorrentURL    string
}

func (c *APIClient) AddTask(params *AddTaskParams) (request.APIResponse[AddTaskData], error) {
	form := url.Values{}
	if params.FolderId != 0 {
		form.Add("folder_id", strconv.FormatInt(params.FolderId, 10))
	}
	if params.TorrentMagnet != "" {
		form.Add("torrent_magnet", params.TorrentMagnet)
	} else if params.TorrentURL != "" {
		form.Add("torrent_url", params.TorrentURL)
	}
	params.Form = &form
	response := &AddTaskData{}
	res, err := c.Request("POST", "/v0.1/p/tasks", params, response)
	return request.NewAPIResponse(res, *response), err
}

type GetTasksDataTorrent struct {
	Id              int64   `json:"id"`
	Name            string  `json:"name"`
	Size            int64   `json:"size"`
	Hash            string  `json:"hash"`
	NodeId          string  `json:"node_id"`
	DownloadRate    int64   `json:"download_rate"`
	TorrentQuality  int     `json:"torrent_quality"`
	ConnectedTo     int     `json:"connected_to"`
	DownloadingFrom int     `json:"downloading_from"`
	UploadingTo     int     `json:"uploading_to"`
	Seeders         int     `json:"seeders"`
	Leechers        int     `json:"leechers"`
	Warnings        []any   `json:"warnings"`
	Stopped         int     `json:"stopped"`
	Progress        float64 `json:"progress"`
	ProgressURL     string  `json:"progress_url"`
	FolderCreatedId int64   `json:"folder_created_id"`
	FolderId        int64   `json:"folder_id"`
	LastUpdate      string  `json:"last_update"` // "2025-11-30 15:20:15",
	Unwanted        string  `json:"unwanted"`
	MustChooseFiles int     `json:"must_choose_files"`
	IsPrivate       bool    `json:"is_private"`
}

func (t GetTasksDataTorrent) GetName() string {
	if t.Name != t.Hash {
		return t.Name
	}
	return ""
}

// NOTE: api response is missing created_at
func (t GetTasksDataTorrent) GetCreatedAt() time.Time {
	created_at, _ := time.Parse(time.DateTime, t.LastUpdate)
	return created_at
}

type GetTasksData struct {
	ResponseContainer
	Torrents []GetTasksDataTorrent `json:"torrents"`
	UserId   int64                 `json:"user_id"`
}

type GetTasksParams struct {
	Ctx
}

func (c *APIClient) GetTasks(params *GetTasksParams) (request.APIResponse[GetTasksData], error) {
	response := &GetTasksData{}
	res, err := c.Request("GET", "/v0.1/p/tasks", params, response)
	return request.NewAPIResponse(res, *response), err
}

type GetTaskData struct {
	ResponseContainer
	Id              int64   `json:"id"`
	Hash            string  `json:"hash"`
	NodeId          string  `json:"node_id"`
	Stopped         int     `json:"stopped"` // 0
	FolderCreatedId int64   `json:"folder_created_id"`
	FolderId        int64   `json:"folder_id"`
	LastUpdate      string  `json:"last_update"` // "2025-11-30 15:20:15",
	Unwanted        string  `json:"unwanted"`
	SpaceMax        int64   `json:"space_max"`
	SpaceUsed       int64   `json:"space_used"`
	Name            string  `json:"name"`
	Type            string  `json:"type"` // "torrent"
	Progress        float64 `json:"progress"`
	Speed           int64   `json:"speed"`
	Size            int64   `json:"size"`
	ProgressURL     string  `json:"progress_url"`
	Parent          int64   `json:"parent"`
	Timestamp       string  `json:"timestamp"` // "2025-11-30 15:20:15"
}

func (t GetTaskData) GetCreatedAt() time.Time {
	created_at, _ := time.Parse(time.DateTime, t.Timestamp)
	return created_at
}

type GetTaskParams struct {
	Ctx
	TaskId int64
}

func (c *APIClient) GetTask(params *GetTaskParams) (request.APIResponse[GetTaskData], error) {
	response := &GetTaskData{}
	res, err := c.Request("GET", "/v0.1/p/tasks/"+strconv.FormatInt(params.TaskId, 10), params, response)
	return request.NewAPIResponse(res, *response), err
}

type GetTaskContentsDataFile struct {
	Name        string `json:"name"` // path/to/file
	Size        int64  `json:"size"`
	Hash        string `json:"hash"`
	LastUpdate  string `json:"last_update"` // "2025-11-30 15:21:34",
	Id          int    `json:"id"`          // 0
	TorrentHash string `json:"torrent_hash"`
}

func (f GetTaskContentsDataFile) GetName() string {
	return filepath.Base(f.Name)
}

func (f GetTaskContentsDataFile) GetPath() string {
	path, _ := util.RemoveRootFolderFromPath(f.Name)
	return path
}

type GetTaskContentsData struct {
	GetTaskData
	Files []GetTaskContentsDataFile `json:"files"`
}

func (t GetTaskContentsData) GetName() string {
	if len(t.Files) > 0 {
		if path, rootFolder := util.RemoveRootFolderFromPath(t.Files[0].Name); rootFolder != "" {
			return rootFolder
		} else {
			return strings.Trim(path, "/")
		}
	}
	if t.Name != t.Hash {
		return t.Name
	}
	return ""
}

type GetTaskContentsParams struct {
	Ctx
	TaskId int64
}

func (c *APIClient) GetTaskContents(params *GetTaskContentsParams) (request.APIResponse[GetTaskContentsData], error) {
	response := &GetTaskContentsData{}
	res, err := c.Request("GET", "/v0.1/p/tasks/"+strconv.FormatInt(params.TaskId, 10)+"/contents", params, response)
	return request.NewAPIResponse(res, *response), err
}

type DeleteTaskData struct {
	ResponseContainer
}

type DeleteTaskParams struct {
	Ctx
	TaskId int64
}

func (c *APIClient) DeleteTask(params *DeleteTaskParams) (request.APIResponse[DeleteTaskData], error) {
	response := &DeleteTaskData{}
	res, err := c.Request("DELETE", "/v0.1/p/tasks/"+strconv.FormatInt(params.TaskId, 10), params, response)
	return request.NewAPIResponse(res, *response), err
}
