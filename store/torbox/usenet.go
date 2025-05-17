package torbox

import (
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type CheckUsenetCachedDataItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

type CheckUsenetCachedData []CheckUsenetCachedDataItem

type CheckUsenetCachedParams struct {
	Ctx
	Hashes []string
}

func (c APIClient) CheckUsenetCached(params *CheckUsenetCachedParams) (APIResponse[CheckUsenetCachedData], error) {
	params.Query = &url.Values{"hash": params.Hashes}
	params.Query.Add("format", "list")
	response := &Response[CheckUsenetCachedData]{}
	res, err := c.Request("GET", "/v1/api/usenet/checkcached", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type CreateUsenetDownloadData struct {
	UsenetDownloadId int    `json:"usenetdownload_id"`
	Hash             string `json:"hash"`
	AuthId           string `json:"auth_id"`
}

type CreateUsenetDownloadParamsPostProcessing int

const (
	CreateUsenetDownloadParamsPostProcessingDefault CreateUsenetDownloadParamsPostProcessing = iota
	CreateUsenetDownloadParamsPostProcessingNone
	CreateUsenetDownloadParamsPostProcessingRepair
	CreateUsenetDownloadParamsPostProcessingRepairUnpack
	CreateUsenetDownloadParamsPostProcessingRepairUnpackDelete
)

type CreateUsenetDownloadParams struct {
	Ctx
	Link           string
	Name           string
	Password       string
	PostProcessing CreateUsenetDownloadParamsPostProcessing
	AsQueued       bool
}

func (c APIClient) CreateUsenetDownload(params *CreateUsenetDownloadParams) (APIResponse[CreateUsenetDownloadData], error) {
	form := &url.Values{}
	form.Add("link", params.Link)
	if params.Name != "" {
		form.Add("name", params.Name)
	}
	if params.Password != "" {
		form.Add("password", params.Password)
	}
	if params.PostProcessing != 0 {
		form.Add("post_processing", strconv.Itoa(int(params.PostProcessing-1)))
	}
	form.Add("as_queued", strconv.FormatBool(params.AsQueued))
	params.Form = form
	response := &Response[CreateUsenetDownloadData]{}
	res, err := c.Request("POST", "/v1/api/usenet/createusenetdownload", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type UsenetDownloadState = TorrentDownloadState

type UsenetDownloadFile struct {
	Id           int    `json:"id"`
	MD5          string `json:"md5"`
	Hash         string `json:"hash"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Zipped       bool   `json:"zipped"`
	S3Path       string `json:"s3_path"`
	Infected     bool   `json:"infected"`
	MimeType     string `json:"mimetype"`
	ShortName    string `json:"short_name"`
	AbsolutePath string `json:"absolute_path"`
}

type UsenetDownload struct {
	Id               int                  `json:"id"`
	Hash             string               `json:"hash"`
	CreatedAt        string               `json:"created_at"`
	UpdatedAt        string               `json:"updated_at"`
	Size             int64                `json:"size"`
	Active           bool                 `json:"active"`
	AuthId           string               `json:"auth_id"`
	DownloadState    UsenetDownloadState  `json:"download_state"`
	Progress         float32              `json:"progress"`
	DownloadSpeed    int                  `json:"download_speed"`
	UploadSpeed      int                  `json:"upload_speed"`
	Name             string               `json:"name"`
	ETA              int                  `json:"eta"`
	Server           int                  `json:"server"`
	TorrentFile      bool                 `json:"torrent_file"`
	ExpiresAt        string               `json:"expires_at"`
	DownloadPresent  bool                 `json:"download_present"`
	DownloadFinished bool                 `json:"download_finished"`
	Files            []UsenetDownloadFile `json:"files"`
	InactiveCheck    int                  `json:"inactive_check"`
	Availability     float32              `json:"availability"`
	OriginalUrl      string               `json:"original_url"` // None
	DownloadId       string               `json:"download_id"`
	Cached           bool                 `json:"cached"`
	CachedAt         string               `json:"cached_at"`
}

func (und UsenetDownload) GetAddedAt() time.Time {
	added_at, err := time.Parse(time.RFC3339, und.CreatedAt)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return added_at.UTC()
}

type ListUsenetDownloadData []UsenetDownload

type ListUsenetDownloadParams struct {
	Ctx
	BypassCache bool
	Offset      int // default: 0
	Limit       int // default: 1000
}

func (c APIClient) ListUsenetDownload(params *ListUsenetDownloadParams) (APIResponse[ListUsenetDownloadData], error) {
	params.Query = &url.Values{}
	params.Query.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	if params.Offset != 0 {
		params.Query.Add("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit != 0 {
		params.Query.Add("limit", strconv.Itoa(params.Limit))
	}
	response := &Response[ListUsenetDownloadData]{}
	res, err := c.Request("GET", "/v1/api/usenet/mylist", params, response)
	if sterr, ok := err.(core.StremThruError); ok && sterr.GetStatusCode() == 404 {
		err = nil
	}
	return newAPIResponse(res, response.Data, response.Detail), err
}

type GetUsenetDownloadData = UsenetDownload

type GetUsenetDownloadParams struct {
	Ctx
	Id          int
	BypassCache bool
}

func (c APIClient) GetUsenetDownload(params *GetUsenetDownloadParams) (APIResponse[GetUsenetDownloadData], error) {
	params.Query = &url.Values{}
	params.Query.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	params.Query.Add("id", strconv.Itoa(params.Id))
	response := &Response[GetUsenetDownloadData]{}
	res, err := c.Request("GET", "/v1/api/usenet/mylist", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type ControlUsenetDownloadOperation string

const (
	ControlUsenetDownloadOperationDelete ControlUsenetDownloadOperation = "delete"
	ControlUsenetDownloadOperationPause  ControlUsenetDownloadOperation = "pause"
	ControlUsenetDownloadOperationResume ControlUsenetDownloadOperation = "resume"
)

type ControlUsenetDownloadParams struct {
	Ctx
	UsenetId  int                            `json:"usenet_id"`
	Operation ControlUsenetDownloadOperation `json:"operation"`
	All       bool                           `json:"all"`
}

type ControlUsenetDownloadData struct {
}

func (c APIClient) ControlUsenetDownload(params *ControlUsenetDownloadParams) (APIResponse[ControlUsenetDownloadData], error) {
	params.JSON = params
	response := &Response[ControlUsenetDownloadData]{}
	res, err := c.Request("POST", "/v1/api/usenet/controlusenetdownload", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type RequestUsenetDownloadLinkParams struct {
	Ctx
	UsenetId int
	FileId   int
	ZipLink  bool
	UserIP   string
	// Redirect bool
}

func (c APIClient) RequestUsenetDownloadLink(params *RequestUsenetDownloadLinkParams) (APIResponse[RequestDownloadLinkData], error) {
	query := &url.Values{}
	query.Add("token", params.APIKey)
	query.Add("usenet_id", strconv.Itoa(params.UsenetId))
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
	res, err := c.Request("GET", "/v1/api/usenet/requestdl", params, response)
	return newAPIResponse(res, RequestDownloadLinkData{Link: response.Data}, response.Detail), err
}
