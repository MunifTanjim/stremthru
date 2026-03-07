package offcloud

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type GetCacheInfoParams struct {
	Ctx
	URLs         []string `json:"urls"`
	IncludeFiles bool     `json:"includeFiles,omitempty"`
}

type GetCacheInfoDataItem struct {
	Cached bool `json:"cached"`
	Files  []struct {
		Folder   []string `json:"folder"`
		Filename string   `json:"filename"`
	} `json:"files,omitempty"`
}

type GetCacheInfoData []GetCacheInfoDataItem

type getCacheInfoData struct {
	ResponseContainer
	data GetCacheInfoData
}

func (c *getCacheInfoData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		c.ResponseContainer = rerr
		return nil
	}

	var items GetCacheInfoData
	if err := json.Unmarshal(data, &items); err == nil {
		c.data = items
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

func (c APIClient) GetCacheInfo(params *GetCacheInfoParams) (APIResponse[GetCacheInfoData], error) {
	params.JSON = params
	response := &getCacheInfoData{}
	res, err := c.Request("POST", "/api/cache/info", params, response)
	return newAPIResponse(res, response.data), err
}

type CloudDownloadStatus string

const (
	CloudDownloadStatusCreated     CloudDownloadStatus = "created"
	CloudDownloadStatusQueued      CloudDownloadStatus = "queued"
	CloudDownloadStatusDownloading CloudDownloadStatus = "downloading"
	CloudDownloadStatusDownloaded  CloudDownloadStatus = "downloaded"
	CloudDownloadStatusError       CloudDownloadStatus = "error"
	CloudDownloadStatusCanceled    CloudDownloadStatus = "canceled"
)

type AddCloudDownloadParams struct {
	Ctx
	URL string `json:"url"`
}

type AddCloudDownloadData struct {
	ResponseContainer
	NotAvailable string `json:"not_available,omitempty"` // 'cloud'

	RequestId    string              `json:"requestId"`
	FileName     string              `json:"fileName"`
	Status       CloudDownloadStatus `json:"status"`
	OriginalLink string              `json:"originalLink"` // e.g. `magnet?:xt=urn:btih:{HASH}`
}

// func (acdd *AddCloudDownloadData) GetServer() string {
// 	info, err := acdd.URL.parse()
// 	if err != nil {
// 		return ""
// 	}
// 	return info.server
// }

func (c APIClient) AddCloudDownload(params *AddCloudDownloadParams) (APIResponse[AddCloudDownloadData], error) {
	params.JSON = params
	response := &AddCloudDownloadData{}
	res, err := c.Request("POST", "/api/cloud", params, response)
	if err == nil && response.NotAvailable != "" {
		response.Err = "not_available: " + response.NotAvailable
		error := UpstreamErrorWithCause(response)
		error.Code = core.ErrorCodeStoreLimitExceeded
		err = error
	}
	return newAPIResponse(res, *response), err
}

type GetCloudDownloadStatusParams struct {
	Ctx
	RequestId string `json:"requestId"`
}

type ExploreCloudDownloadParams struct {
	Ctx
	RequestId string
}

type ExploreCloudDownloadDataFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

func (f *ExploreCloudDownloadDataFile) GetName() string {
	return filepath.Base(f.Path)
}

func (f *ExploreCloudDownloadDataFile) GetPath() string {
	path, _ := util.RemoveRootFolderFromPath(f.Path)
	return path
}

type ExploreCloudDownloadData struct {
	ResponseContainer
	Files []ExploreCloudDownloadDataFile `json:"files"`
}

func (c APIClient) ExploreCloudDownload(params *ExploreCloudDownloadParams) (APIResponse[ExploreCloudDownloadData], error) {
	params.Query = &url.Values{
		"format": {"detailed"},
	}
	response := ExploreCloudDownloadData{}
	res, err := c.Request("GET", "/api/cloud/explore/"+params.RequestId, params, &response)
	return newAPIResponse(res, response), err
}

type GetCloudHistoryParams struct {
	Ctx
}

type GetCloudHistoryDataItem struct {
	RequestId    string              `json:"requestId"`
	FileName     string              `json:"fileName"`
	Status       CloudDownloadStatus `json:"status"`
	OriginalLink string              `json:"originalLink"`
	CreatedOn    time.Time           `json:"createdOn"`
}

type GetCloudHistoryData []GetCloudHistoryDataItem

type getCloudHistoryData struct {
	ResponseContainer
	data GetCloudHistoryData
}

func (d *getCloudHistoryData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		d.ResponseContainer = rerr
		return nil
	}

	var items GetCloudHistoryData
	if err := json.Unmarshal(data, &items); err == nil {
		d.data = items
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

func (c APIClient) GetCloudHistory(params *GetCloudHistoryParams) (APIResponse[GetCloudHistoryData], error) {
	response := getCloudHistoryData{}
	res, err := c.Request("GET", "/api/cloud/history", params, &response)
	return newAPIResponse(res, response.data), err
}
