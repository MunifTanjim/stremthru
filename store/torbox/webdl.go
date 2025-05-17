package torbox

import (
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type CheckWebDLCachedDataItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

type CheckWebDLCachedData []CheckWebDLCachedDataItem

type CheckWebDLCachedParams struct {
	Ctx
	Hashes []string
}

func (c APIClient) CheckWebDLCached(params *CheckWebDLCachedParams) (APIResponse[CheckWebDLCachedData], error) {
	params.Query = &url.Values{"hash": params.Hashes}
	params.Query.Add("format", "list")
	response := &Response[CheckWebDLCachedData]{}
	res, err := c.Request("GET", "/v1/api/webdl/checkcached", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type CreateWebDLDownloadData struct {
	UsenetDownloadId int    `json:"webdownload_id"`
	Hash             string `json:"hash"`
	AuthId           string `json:"auth_id"`
}

type CreateWebDLDownloadParamsPostProcessing int

const (
	CreateWebDLDownloadParamsPostProcessingDefault CreateWebDLDownloadParamsPostProcessing = iota
	CreateWebDLDownloadParamsPostProcessingNone
	CreateWebDLDownloadParamsPostProcessingRepair
	CreateWebDLDownloadParamsPostProcessingRepairUnpack
	CreateWebDLDownloadParamsPostProcessingRepairUnpackDelete
)

type CreateWebDLDownloadParams struct {
	Ctx
	Link     string
	Name     string
	Password string
	AsQueued bool
}

func (c APIClient) CreateWebDLDownload(params *CreateWebDLDownloadParams) (APIResponse[CreateWebDLDownloadData], error) {
	form := &url.Values{}
	form.Add("link", params.Link)
	if params.Name != "" {
		form.Add("name", params.Name)
	}
	if params.Password != "" {
		form.Add("password", params.Password)
	}
	form.Add("as_queued", strconv.FormatBool(params.AsQueued))
	params.Form = form
	response := &Response[CreateWebDLDownloadData]{}
	res, err := c.Request("POST", "/v1/api/webdl/createwebdownload", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type WebDLDownloadState = TorrentDownloadState // decrypting

type WebDLDownloadFile struct {
	AbsolutePath string `json:"absolute_path"`
	Hash         string `json:"hash"`
	Id           int    `json:"id"`
	Infected     bool   `json:"infected"`
	MD5          string `json:"md5"` // null
	MimeType     string `json:"mimetype"`
	Name         string `json:"name"`
	S3Path       string `json:"s3_path"`
	ShortName    string `json:"short_name"`
	Size         int64  `json:"size"`
	Zipped       bool   `json:"zipped"`
}

type WebDLDownload struct {
	Active           bool                `json:"active"`
	AuthId           string              `json:"auth_id"`
	Availability     float32             `json:"availability"`
	Cached           bool                `json:"cached"`
	CachedAt         string              `json:"cached_at"`
	CreatedAt        string              `json:"created_at"`
	DownloadFinished bool                `json:"download_finished"`
	DownloadPresent  bool                `json:"download_present"`
	DownloadSpeed    int                 `json:"download_speed"`
	DownloadState    WebDLDownloadState  `json:"download_state"`
	ETA              int                 `json:"eta"`
	ExpiresAt        string              `json:"expires_at"`
	Files            []WebDLDownloadFile `json:"files"`
	Hash             string              `json:"hash"`
	Id               int                 `json:"id"`
	InactiveCheck    int                 `json:"inactive_check"`
	Name             string              `json:"name"`
	OriginalUrl      string              `json:"original_url"` // None
	Progress         float32             `json:"progress"`
	Server           int                 `json:"server"`
	Size             int64               `json:"size"`
	TorrentFile      bool                `json:"torrent_file"`
	UpdatedAt        string              `json:"updated_at"`
	UploadSpeed      int                 `json:"upload_speed"`
}

func (und WebDLDownload) GetAddedAt() time.Time {
	added_at, err := time.Parse(time.RFC3339, und.CreatedAt)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return added_at.UTC()
}

type ListWebDLDownloadData []WebDLDownload

type ListWebDLDownloadParams struct {
	Ctx
	BypassCache bool
	Offset      int // default: 0
	Limit       int // default: 1000
}

func (c APIClient) ListWebDLDownload(params *ListWebDLDownloadParams) (APIResponse[ListWebDLDownloadData], error) {
	params.Query = &url.Values{}
	params.Query.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	if params.Offset != 0 {
		params.Query.Add("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit != 0 {
		params.Query.Add("limit", strconv.Itoa(params.Limit))
	}
	response := &Response[ListWebDLDownloadData]{}
	res, err := c.Request("GET", "/v1/api/webdl/mylist", params, response)
	if sterr, ok := err.(core.StremThruError); ok && sterr.GetStatusCode() == 404 {
		err = nil
	}
	return newAPIResponse(res, response.Data, response.Detail), err
}

type GetWebDLDownloadData = WebDLDownload

type GetWebDLDownloadParams struct {
	Ctx
	Id          int
	BypassCache bool
}

func (c APIClient) GetWebDLDownload(params *GetWebDLDownloadParams) (APIResponse[GetWebDLDownloadData], error) {
	params.Query = &url.Values{}
	params.Query.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	params.Query.Add("id", strconv.Itoa(params.Id))
	response := &Response[GetWebDLDownloadData]{}
	res, err := c.Request("GET", "/v1/api/webdl/mylist", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type ControlWebDLDownloadOperation string

const (
	ControlWebDLDownloadOperationDelete ControlWebDLDownloadOperation = "delete"
)

type ControlWebDLDownloadParams struct {
	Ctx
	WebDLId   int                           `json:"webdl_id"`
	Operation ControlWebDLDownloadOperation `json:"operation"`
	All       bool                          `json:"all"`
}

type ControlWebDLDownloadData struct {
}

func (c APIClient) ControlWebDLDownload(params *ControlWebDLDownloadParams) (APIResponse[ControlWebDLDownloadData], error) {
	params.JSON = params
	response := &Response[ControlWebDLDownloadData]{}
	res, err := c.Request("POST", "/v1/api/webdl/controlwebdownload", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type RequestWebDLDownloadLinkParams struct {
	Ctx
	WebDLId int
	FileId  int
	ZipLink bool
	UserIP  string
	// Redirect bool
}

func (c APIClient) RequestWebDLDownloadLink(params *RequestWebDLDownloadLinkParams) (APIResponse[RequestDownloadLinkData], error) {
	query := &url.Values{}
	query.Add("token", params.APIKey)
	query.Add("web_id", strconv.Itoa(params.WebDLId))
	if params.FileId != 0 {
		query.Add("file_id", strconv.Itoa(params.FileId))
	}
	query.Add("zip_link", strconv.FormatBool(params.ZipLink))
	if params.UserIP != "" {
		query.Add("user_ip", params.UserIP)
	}
	// if params.Redirect {
	// 	query.Add("redirect", strconv.FormatBool(params.Redirect))
	// }
	params.Query = query
	response := &Response[string]{}
	res, err := c.Request("GET", "/v1/api/webdl/requestdl", params, response)
	return newAPIResponse(res, RequestDownloadLinkData{Link: response.Data}, response.Detail), err
}
