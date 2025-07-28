package debrider

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type DownloadTaskType string

const (
	DownloadTaskTypeMagnet  DownloadTaskType = "magnet"
	DownloadTaskTypeNzb     DownloadTaskType = "nzb"
	DownloadTaskTypeTorrent DownloadTaskType = "torrent"
	DownloadTaskTypeWeb     DownloadTaskType = "web"
)

type CreateDownloadTaskData struct {
	ResponseContainer
	Data Task `json:"data"`
}

type createWebDownloadTaskParamsData struct {
	Url      string `json:"url"`
	Password string `json:"password,omitempty"`
}

type CreateDownloadTaskParamsData struct {
	FileContent string // nzb / torrent
	MagnetLink  string // magnet
	Url         string // web
	Password    string // web
}

func (d *CreateDownloadTaskParamsData) MarshalJSON() ([]byte, error) {
	if d.FileContent != "" {
		return json.Marshal(core.Base64EncodeToByte(d.FileContent))
	}
	if d.MagnetLink != "" {
		return json.Marshal(d.MagnetLink)
	}
	if d.Url != "" {
		return json.Marshal(createWebDownloadTaskParamsData{
			Url:      d.Url,
			Password: d.Password,
		})
	}
	return nil, core.NewError("no content to marshal")
}

type CreateDownloadTaskParams struct {
	Ctx
	Type DownloadTaskType             `json:"type"`
	Data CreateDownloadTaskParamsData `json:"data"`
}

func (c APIClient) CreateDownloadTask(params *CreateDownloadTaskParams) (APIResponse[Task], error) {
	params.JSON = params

	response := CreateDownloadTaskData{}
	res, err := c.Request("POST", "/v1/tasks", params, &response)
	return newAPIResponse(res, response.Data), err
}

type TaskFile struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	DownloadLink string `json:"download_link,omitempty"`
}

func (f *TaskFile) GetPath() string {
	return "/" + f.Name
}

func (f *TaskFile) GetName() string {
	return filepath.Base(f.GetPath())
}

type TaskStatus string

const (
	TaskStatusError       TaskStatus = "error"
	TaskStatusParsing     TaskStatus = "parsing"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusCompleted   TaskStatus = "completed"
)

type Task struct {
	Id            string     `json:"id"`
	Hash          string     `json:"hash"`
	Name          string     `json:"name"`
	Size          int64      `json:"size"`
	Files         []TaskFile `json:"files"`
	Progress      float64    `json:"progress"`
	Status        TaskStatus `json:"status"`
	DownloadSpeed int64      `json:"downloadSpeed"` // in bytes per second
	UploadSpeed   int64      `json:"uploadSpeed"`   // in bytes per second
	ETA           int64      `json:"eta"`           // in seconds
	AddedDate     string     `json:"addedDate"`
	Type          string     `json:"type"` // torrent
}

func (li *Task) GetAddedAt() time.Time {
	t, err := time.Parse(time.RFC3339, li.AddedDate)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return t.UTC()
}

type ListTaskData []Task

type listTaskData struct {
	ResponseContainer
	data ListTaskData
}

func (l *listTaskData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer
	err := json.Unmarshal(data, &rerr)
	if err == nil {
		l.ResponseContainer = rerr
		return nil
	}

	var items ListTaskData
	err = core.UnmarshalJSON(200, data, &items)
	if err == nil {
		l.data = items
		return nil
	}

	e := core.NewAPIError("failed to parse response")
	e.Cause = err
	return e
}

type ListTaskParams struct {
	Ctx
}

func (c APIClient) ListTask(params *ListTaskParams) (APIResponse[ListTaskData], error) {
	response := &listTaskData{}
	res, err := c.Request("GET", "/v1/tasks", params, response)
	return newAPIResponse(res, response.data), err
}

type DeleteTaskData struct {
	ResponseContainer
}

type DeleteTaskParams struct {
	Ctx
	Id string
}

func (c APIClient) DeleteTask(params *DeleteTaskParams) (APIResponse[DeleteTaskData], error) {
	response := &DeleteTaskData{}
	res, err := c.Request("DELETE", "/v1/tasks/"+params.Id, params, response)
	return newAPIResponse(res, *response), err
}

type GetTaskData struct {
	ResponseContainer
	Task
}

type GetTaskParams struct {
	Ctx
	Id string
}

func (c APIClient) GetTask(params *GetTaskParams) (APIResponse[Task], error) {
	response := &GetTaskData{}
	res, err := c.Request("GET", "/v1/tasks/"+params.Id, params, response)
	return newAPIResponse(res, response.Task), err
}
