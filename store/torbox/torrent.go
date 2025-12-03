package torbox

import (
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type CheckTorrentsCachedDataItemFile struct {
	Name string `json:"name"` // full path
	Size int64  `json:"size"`
}

func (f CheckTorrentsCachedDataItemFile) GetName() string {
	return filepath.Base(f.Name)
}

func (f CheckTorrentsCachedDataItemFile) GetPath() string {
	path, _ := util.RemoveRootFolderFromPath(f.Name)
	return path
}

type CheckTorrentsCachedDataItem struct {
	Name     string                            `json:"name"`
	Size     int64                             `json:"size"`
	Hash     string                            `json:"hash"`
	Trackers []any                             `json:"trackers"`
	Seeds    int                               `json:"seeds"`
	Peers    int                               `json:"peers"`
	Files    []CheckTorrentsCachedDataItemFile `json:"files"`
}

func (t CheckTorrentsCachedDataItem) GetName() string {
	if len(t.Files) > 0 {
		if _, rootFolder := util.RemoveRootFolderFromPath(t.Files[0].Name); rootFolder != "" {
			return rootFolder
		}
	}
	return t.Name
}

type CheckTorrentsCachedData []CheckTorrentsCachedDataItem

type CheckTorrentsCachedParams struct {
	Ctx
	Hashes    []string
	ListFiles bool
}

func (c APIClient) CheckTorrentsCached(params *CheckTorrentsCachedParams) (APIResponse[CheckTorrentsCachedData], error) {
	form := &url.Values{"hash": params.Hashes}
	form.Add("format", "list")
	form.Add("list_files", strconv.FormatBool(params.ListFiles))
	params.Form = form
	response := &Response[CheckTorrentsCachedData]{}
	res, err := c.Request("GET", "/v1/api/torrents/checkcached", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type CreateTorrentData struct {
	TorrentId int    `json:"torrent_id"`
	Hash      string `json:"hash"`
	AuthId    string `json:"auth_id"`
}

type CreateTorrentParamsSeed int

const (
	CreateTorrentParamsSeedAuto CreateTorrentParamsSeed = 1
	CreateTorrentParamsSeedYes  CreateTorrentParamsSeed = 2
	CreateTorrentParamsSeedNo   CreateTorrentParamsSeed = 3
)

type CreateTorrentParams struct {
	Ctx
	File     *multipart.FileHeader
	Magnet   string
	Seed     int
	AllowZip bool
	Name     string
}

/*
Possible Detail values:
  - Found Cached Torrent. Using Cached Torrent.
*/
func (c APIClient) CreateTorrent(params *CreateTorrentParams) (APIResponse[CreateTorrentData], error) {
	if params.Magnet != "" {
		form := &url.Values{}
		form.Add("magnet", params.Magnet)
		if params.Seed != 0 {
			form.Add("seed", strconv.Itoa(int(params.Seed)))
		}
		form.Add("allow_zip", strconv.FormatBool(params.AllowZip))
		if params.Name != "" {
			form.Add("name", params.Name)
		}
		params.Form = form
	} else {
		form := &multipart.Form{}
		form.File = map[string][]*multipart.FileHeader{
			"file": {params.File},
		}
		form.Value = map[string][]string{
			"allow_zip": {strconv.FormatBool(params.AllowZip)},
		}
		if params.Seed != 0 {
			form.Value["seed"] = []string{strconv.Itoa(int(params.Seed))}
		}
		if params.Name != "" {
			form.Value["name"] = []string{params.Name}
		}
		params.MultiPartForm = form
	}
	response := &Response[CreateTorrentData]{}
	res, err := c.Request("POST", "/v1/api/torrents/createtorrent", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type TorrentFile struct {
	Id                int    `json:"id"`
	MD5               string `json:"md5"`
	Hash              string `json:"hash"`
	Name              string `json:"name"`
	Size              int64  `json:"size"`
	Zipped            bool   `json:"zipped"`
	S3Path            string `json:"s3_path"`
	Infected          bool   `json:"infected"`
	MimeType          string `json:"mimetype"`
	ShortName         string `json:"short_name"`
	AbsolutePath      string `json:"absolute_path"`
	OpensubtitlesHash string `json:"opensubtitles_hash"`
}

func (f TorrentFile) GetName() string {
	return f.ShortName
}

func (f TorrentFile) GetPath() string {
	path, _ := util.RemoveRootFolderFromPath(f.Name)
	return path
}

type TorrentDownloadState string

const (
	TorrentDownloadStateDownloading        TorrentDownloadState = "downloading"        // The torrent is currently downloading
	TorrentDownloadStateUploading          TorrentDownloadState = "uploading"          // The torrent is currently seeding
	TorrentDownloadStateStalled            TorrentDownloadState = "stalled"            // The torrent is trying to download, but there are no seeds connected to download from
	TorrentDownloadStatePaused             TorrentDownloadState = "paused"             // The torrent is paused
	TorrentDownloadStateCompleted          TorrentDownloadState = "completed"          // The torrent is completely downloaded. Do not use this for download completion status
	TorrentDownloadStateCached             TorrentDownloadState = "cached"             // The torrent is cached from the server
	TorrentDownloadStateMetaDL             TorrentDownloadState = "metaDL"             // The torrent is downloading metadata from the hoard
	TorrentDownloadStateCheckingResumeData TorrentDownloadState = "checkingResumeData" // The torrent is checking resumable data
)

type Torrent struct {
	Id               int                  `json:"id"`
	Hash             string               `json:"hash"`
	CreatedAt        string               `json:"created_at"`
	UpdatedAt        string               `json:"updated_at"`
	Magnet           string               `json:"magnet"`
	Size             int64                `json:"size"`
	Active           bool                 `json:"active"`
	AuthId           string               `json:"auth_id"`
	DownloadState    TorrentDownloadState `json:"download_state"`
	Seeds            int                  `json:"seeds"`
	Peers            int                  `json:"peers"`
	Ratio            float32              `json:"ratio"`
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
	Files            []TorrentFile        `json:"files"`
	InactiveCheck    int                  `json:"inactive_check"`
	Availability     float32              `json:"availability"`
	Owner            string               `json:"owner"`
	Private          bool                 `json:"private"`
	// AllowZipped      bool                 `json:"allow_zipped"`
	// Cached           bool                 `json:"cached"`
	// CachedAt         string               `json:"cached_at"`
	// DownloadPath     string               `json:"download_path"`
	// LongTermSeeding  bool                 `json:"long_term_seeding"`
	// SeedTorrent      bool                 `json:"seed_torrent"`
	// TotalDownloaded  int64                `json:"total_downloaded"`
	// TotalUploaded    int64                `json:"total_uploaded"`
	// Tracker          string               `json:"tracker"`
	// TrackerMessage   any                  `json:"tracker_message"`
}

func (t Torrent) GetName() string {
	if len(t.Files) > 0 {
		if _, rootFolder := util.RemoveRootFolderFromPath(t.Files[0].Name); rootFolder != "" {
			return rootFolder
		}
	}
	return t.Name
}

func (t Torrent) GetAddedAt() time.Time {
	added_at, err := time.Parse(time.RFC3339, t.CreatedAt)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return added_at.UTC()
}

type ListTorrentsData []Torrent

type ListTorrentsParams struct {
	Ctx
	BypassCache bool
	Offset      int // default: 0
	Limit       int // default: 1000
}

func (c APIClient) ListTorrents(params *ListTorrentsParams) (APIResponse[ListTorrentsData], error) {
	form := &url.Values{}
	form.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	if params.Offset != 0 {
		form.Add("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit != 0 {
		form.Add("limit", strconv.Itoa(params.Limit))
	}
	params.Form = form
	response := &Response[ListTorrentsData]{}
	res, err := c.Request("GET", "/v1/api/torrents/mylist", params, response)
	if sterr, ok := err.(core.StremThruError); ok && sterr.GetStatusCode() == 404 {
		err = nil
	}
	return newAPIResponse(res, response.Data, response.Detail), err
}

type GetTorrentData = Torrent

type GetTorrentParams struct {
	Ctx
	Id          int
	BypassCache bool
}

func (c APIClient) GetTorrent(params *GetTorrentParams) (APIResponse[GetTorrentData], error) {
	form := &url.Values{}
	form.Add("bypass_cache", strconv.FormatBool(params.BypassCache))
	form.Add("id", strconv.Itoa(params.Id))
	params.Form = form
	response := &Response[GetTorrentData]{}
	res, err := c.Request("GET", "/v1/api/torrents/mylist", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type ControlTorrentOperation string

const (
	ControlTorrentOperationReannounce ControlTorrentOperation = "reannounce"
	ControlTorrentOperationDelete     ControlTorrentOperation = "delete"
	ControlTorrentOperationResume     ControlTorrentOperation = "resume"
	ControlTorrentOperationPause      ControlTorrentOperation = "pause"
)

type ControlTorrentParams struct {
	Ctx
	TorrentId int                     `json:"torrent_id"`
	Operation ControlTorrentOperation `json:"operation"`
	All       bool                    `json:"all"`
}

type ControlTorrentData struct {
}

func (c APIClient) ControlTorrent(params *ControlTorrentParams) (APIResponse[ControlTorrentData], error) {
	params.JSON = params
	response := &Response[ControlTorrentData]{}
	res, err := c.Request("POST", "/v1/api/torrents/controltorrent", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}

type RequestDownloadLinkData struct {
	Link string
}

type RequestDownloadLinkParams struct {
	Ctx
	TorrentId int
	FileId    int
	ZipLink   bool
	UserIP    string
	// Redirect  bool
}

func (c APIClient) RequestDownloadLink(params *RequestDownloadLinkParams) (APIResponse[RequestDownloadLinkData], error) {
	query := &url.Values{}
	query.Add("token", params.APIKey)
	query.Add("torrent_id", strconv.Itoa(params.TorrentId))
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
	res, err := c.Request("GET", "/v1/api/torrents/requestdl", params, response)
	return newAPIResponse(res, RequestDownloadLinkData{Link: response.Data}, response.Detail), err
}

type GetTorrentInfoData = CheckTorrentsCachedDataItem

type GetTorrentInfoParams struct {
	Ctx
	Hash    string
	Timeout int // default: 10
}

func (c APIClient) GetTorrentInfo(params *GetTorrentInfoParams) (APIResponse[GetTorrentInfoData], error) {
	form := &url.Values{}
	form.Add("hash", params.Hash)
	if params.Timeout != 0 {
		form.Add("timeout", strconv.Itoa(params.Timeout))
	}
	params.Form = form
	response := &Response[GetTorrentInfoData]{}
	res, err := c.Request("GET", "/v1/api/torrents/torrentinfo", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}
