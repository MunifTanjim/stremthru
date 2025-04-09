package offcloud

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type CheckCacheParams struct {
	Ctx
	Hashes []string `json:"hashes"`
}

type CheckCacheData struct {
	ResponseContainer
	CachedItems []string `json:"cachedItems"`
}

func (c APIClient) CheckCache(params *CheckCacheParams) (APIResponse[CheckCacheData], error) {
	c.injectAPIKey(&params.Ctx)
	params.JSON = params
	response := &CheckCacheData{}
	res, err := c.Request("POST", "/api/cache", params, response)
	return newAPIResponse(res, *response), err
}

type CloudDownloadStatus string

const (
	CloudDownloadStatusCreated    CloudDownloadStatus = "created"
	CloudDownloadStatusDownloaded CloudDownloadStatus = "downloaded"
	CloudDownloadStatusError      CloudDownloadStatus = "error"
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
	Site         string              `json:"site"`
	Status       CloudDownloadStatus `json:"status"`
	OriginalLink string              `json:"originalLink"` // e.g. `magnet?:xt=urn:btih:{HASH}`
	URL          CloudDownloadLink   `json:"url"`          // e.g. `https://{SERVER}.offlcoud.com/cloud/download/{REQUEST_ID}`
	CreatedOn    time.Time           `json:"createdOn"`
}

func (acdd *AddCloudDownloadData) GetServer() string {
	info, err := acdd.URL.parse()
	if err != nil {
		return ""
	}
	return info.server
}

func (c APIClient) AddCloudDownload(params *AddCloudDownloadParams) (APIResponse[AddCloudDownloadData], error) {
	c.injectAPIKey(&params.Ctx)
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

type GetCloudDownloadStatusDataStatus struct {
	Status      CloudDownloadStatus `json:"status"`
	Amount      int                 `json:"amount"`
	RequestId   string              `json:"requestId"`
	FileName    string              `json:"fileName"` // can be renamed
	FileSize    int64               `json:"fileSize"`
	Server      string              `json:"server"`
	IsDirectory bool                `json:"isDirectory"`
}

type GetCloudDownloadStatusData struct {
	ResponseContainer
	Status GetCloudDownloadStatusDataStatus `json:"status"`
}

func (c APIClient) GetCloudDownloadStatus(params *GetCloudDownloadStatusParams) (APIResponse[GetCloudDownloadStatusData], error) {
	c.injectAPIKey(&params.Ctx)
	params.JSON = params
	response := &GetCloudDownloadStatusData{}
	res, err := c.Request("POST", "/api/cloud/status", params, response)
	return newAPIResponse(res, *response), err
}

type ExploreCloudDownloadParams struct {
	Ctx
	RequestId string `json:"requestId"`
}

type CloudDownloadLink string // e.g. `https://{SERVER}.offlcoud.com/cloud/download/{REQUEST_ID}/{FILE_IDX}/{FILE_NAME}`

type parsedCloudDownloadLinkInfo struct {
	server    string
	path      string
	requestId string
	fileIdx   int
	fileName  string
}

func (link CloudDownloadLink) parse() (*parsedCloudDownloadLinkInfo, error) {
	u, err := url.Parse(string(link))
	if err != nil {
		return nil, err
	}
	info := &parsedCloudDownloadLinkInfo{}
	server, _, _ := strings.Cut(u.Host, ".")
	info.server = server
	info.path = u.Path
	pathParts := strings.Split(info.path, "/")
	info.requestId = pathParts[3]
	if len(pathParts) > 4 {
		fileIdx, err := strconv.Atoi(pathParts[4])
		if err != nil {
			return nil, err
		}
		info.fileIdx = fileIdx
		info.fileName = pathParts[5]
	}
	return info, nil
}

type ExploreCloudDownloadData []CloudDownloadLink // e.g. `https://{SERVER}.offlcoud.com/cloud/download/{REQUEST_ID}/{FILE_IDX}/{FILE_NAME}`

type exploreCloudDownloadData struct {
	ResponseContainer
	data ExploreCloudDownloadData
}

func (c *exploreCloudDownloadData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		c.ResponseContainer = rerr
		return nil
	}

	var items ExploreCloudDownloadData
	if err := json.Unmarshal(data, &items); err == nil {
		c.data = items
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

func (c APIClient) ExploreCloudDownload(params *ExploreCloudDownloadParams) (APIResponse[ExploreCloudDownloadData], error) {
	c.injectAPIKey(&params.Ctx)
	params.JSON = params
	response := &exploreCloudDownloadData{}
	res, err := c.Request("POST", "/api/cloud/explore", params, response)
	return newAPIResponse(res, response.data), err
}

type ListCloudDownloadsParams struct {
	Ctx
	Page int `json:"page"`
}

type ListCloudDownloadsDataItem struct {
	CreatedOn    time.Time           `json:"createdOn"`
	FileName     string              `json:"fileName"`
	FileSize     int64               `json:"fileSize"`
	IsDirectory  bool                `json:"isDirectory"`
	OriginalLink string              `json:"originalLink"`
	RequestId    string              `json:"requestId"`
	Server       string              `json:"server"`
	Site         string              `json:"site"` // 'BitTorrent'
	Status       CloudDownloadStatus `json:"status"`
	UserId       string              `json:"userId"`
}

type ListCloudDownloadsData struct {
	ResponseContainer
	History []ListCloudDownloadsDataItem
	IsEnd   bool
}

func (c APIClient) ListCloudDownloads(params *ListCloudDownloadsParams) (APIResponse[ListCloudDownloadsData], error) {
	c.injectSessionCookie(&params.Ctx)
	params.JSON = params
	response := &ListCloudDownloadsData{}
	res, err := c.Request("POST", "/cloud/history", params, response)
	return newAPIResponse(res, *response), err
}

type ListCloudDownloadEntriesParams struct {
	Ctx
	RequestId string
	Server    string
}

type ListCloudDownloadEntriesData struct {
	ResponseContainer
	Entries     []string `json:"entries"` // e.g. `{MAGNET_NAME}/{FILE_PATH}`, `/{MAGNET_NAME}.aria2`
	File        string   `json:"file"`    // e.g. `{NORMALIZED_MAGNET_NAME}`
	IsDirectory bool     `json:"isDirectory"`
	Server      string   `json:"server"`
}

func (c APIClient) ListCloudDownloadEntries(params *ListCloudDownloadEntriesParams) (APIResponse[ListCloudDownloadEntriesData], error) {
	c.injectSessionCookie(&params.Ctx)
	params.JSON = params
	response := &ListCloudDownloadEntriesData{}

	path := c.BaseURL.JoinPath("/cloud/list")
	path.Host = params.Server + "." + path.Host
	req, err := params.NewRequest(path, "GET", params.RequestId, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameOffcloud)
		error.Cause = err
		return newAPIResponse(nil, *response), error
	}

	res, err := c.doRequest(req, response)
	return newAPIResponse(res, *response), err
}

type RemoveCloudDownloadParams struct {
	Ctx
	RequestId string
}

type RemoveCloudDownloadData struct {
	ResponseContainer
	Success bool `json:"success"`
}

func (c APIClient) RemoveCloudDownload(params *RemoveCloudDownloadParams) (APIResponse[RemoveCloudDownloadData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &RemoveCloudDownloadData{}
	res, err := c.Request("GET", "/cloud/remove/"+params.RequestId, params, response)
	return newAPIResponse(res, *response), err
}
