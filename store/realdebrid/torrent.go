package realdebrid

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type CheckTorrentInstantAvailabilityDataFileIdsVariantFile struct {
	Filename string `json:"filename"`
	Filesize int    `json:"filesize"`
}

type CheckTorrentInstantAvailabilityDataFileIdsVariant = map[string]CheckTorrentInstantAvailabilityDataFileIdsVariantFile

type CheckTorrentInstantAvailabilityDataHosterMap = map[string][]CheckTorrentInstantAvailabilityDataFileIdsVariant

type CheckTorrentInstantAvailabilityData = map[string]CheckTorrentInstantAvailabilityDataHosterMap

type checkTorrentInstantAvailabilityData struct {
	*ResponseError
	data CheckTorrentInstantAvailabilityData
}

func (c *checkTorrentInstantAvailabilityData) UnmarshalJSON(data []byte) error {
	temp := map[string]any{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	isErrorResponse := false
	if err, ok := temp["error"].(string); ok {
		isErrorResponse = true
		if c.ResponseError == nil {
			c.ResponseError = &ResponseError{}
		}
		c.Err = err
	}
	if err_code, ok := temp["error_code"].(float64); ok {
		isErrorResponse = true
		if c.ResponseError == nil {
			c.ResponseError = &ResponseError{}
		}
		c.ErrCode = ErrorCode(err_code)
	}
	if isErrorResponse {
		return nil
	}

	delete(temp, "err")
	delete(temp, "error_code")

	c.data = make(CheckTorrentInstantAvailabilityData)
	for key, value := range temp {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if jsonBytes[0] == '[' {
			jsonBytes = []byte("{}")
		}

		var hosterMap CheckTorrentInstantAvailabilityDataHosterMap
		if err := json.Unmarshal(jsonBytes, &hosterMap); err != nil {
			return err
		}

		c.data[key] = hosterMap
	}

	return nil
}

type CheckTorrentInstantAvailabilityParams struct {
	Ctx
	Hashes []string
}

func (c APIClient) CheckTorrentInstantAvailability(params *CheckTorrentInstantAvailabilityParams) (APIResponse[CheckTorrentInstantAvailabilityData], error) {
	response := &checkTorrentInstantAvailabilityData{}
	res, err := c.Request("GET", "/rest/1.0/torrents/instantAvailability/"+strings.Join(params.Hashes, "/"), params, response)
	return newAPIResponse(res, response.data), err
}

type AddMagnetData struct {
	*ResponseError
	Id  string `json:"id"`
	Uri string `json:"uri"`
}

type AddMagnetParams struct {
	Ctx
	Magnet string
	Host   string
	IP     string
}

func (c APIClient) AddMagnet(params *AddMagnetParams) (APIResponse[AddMagnetData], error) {
	form := &url.Values{}
	form.Add("magnet", params.Magnet)
	if params.Host != "" {
		form.Add("host", params.Host)
	}
	if params.IP != "" {
		form.Add("ip", params.IP)
	}
	params.Form = form
	response := &AddMagnetData{}
	res, err := c.Request("POST", "/rest/1.0/torrents/addMagnet", params, response)
	return newAPIResponse(res, *response), err
}

type TorrentStatus string

const (
	TorrentStatusMagnetError           TorrentStatus = "magnet_error"
	TorrentStatusMagnetConversion      TorrentStatus = "magnet_conversion"
	TorrentStatusWaitingFilesSelection TorrentStatus = "waiting_files_selection"
	TorrentStatusQueued                TorrentStatus = "queued"
	TorrentStatusDownloading           TorrentStatus = "downloading"
	TorrentStatusDownloaded            TorrentStatus = "downloaded"
	TorrentStatusError                 TorrentStatus = "error"
	TorrentStatusVirus                 TorrentStatus = "virus"
	TorrentStatusCompressing           TorrentStatus = "compressing"
	TorrentStatusUploading             TorrentStatus = "uploading"
	TorrentStatusDead                  TorrentStatus = "dead"
)

type GetTorrentInfoDataFile struct {
	Id       int    `json:"id"`       // File unique identifier
	Path     string `json:"path"`     // Path to the file inside the torrent, starting with "/"
	Bytes    int64  `json:"bytes"`    // Size of the file in bytes
	Selected int    `json:"selected"` // Whether the file is selected (0 or 1)
}

type GetTorrentInfoData struct {
	*ResponseError
	Id               string                   `json:"id"`                // Torrent unique identifier
	Filename         string                   `json:"filename"`          // Name of the torrent file
	OriginalFilename string                   `json:"original_filename"` // Original name of the torrent
	Hash             string                   `json:"hash"`              // SHA1 hash of the torrent
	Bytes            int64                    `json:"bytes"`             // Size of selected files only
	OriginalBytes    int64                    `json:"original_bytes"`    // Total size of the torrent
	Host             string                   `json:"host"`              // Host main domain
	Split            int                      `json:"split"`             // Split size of links
	Progress         float32                  `json:"progress"`          // Possible values: 0 to 100
	Status           TorrentStatus            `json:"status"`            // Current status of the torrent
	Added            string                   `json:"added"`             // Date added (jsonDate format)
	Files            []GetTorrentInfoDataFile `json:"files"`             // List of files in the torrent
	Links            []string                 `json:"links"`             // List of host URLs
	Ended            string                   `json:"ended,omitempty"`   // Date ended, only present when finished (jsonDate format)
	Speed            int64                    `json:"speed,omitempty"`   // Speed, only present in specific statuses
	Seeders          int                      `json:"seeders,omitempty"` // Seeders, only present in specific statuses
}

func toAddedAt(added string) time.Time {
	added_at, err := time.Parse(time.RFC3339, added)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return added_at.UTC()

}

func (t GetTorrentInfoData) GetAddedAt() time.Time {
	return toAddedAt(t.Added)
}

type GetTorrentInfoParams struct {
	Ctx
	Id string
}

func (c APIClient) GetTorrentInfo(params *GetTorrentInfoParams) (APIResponse[GetTorrentInfoData], error) {
	response := &GetTorrentInfoData{}
	res, err := c.Request("GET", "/rest/1.0/torrents/info/"+params.Id, params, response)
	return newAPIResponse(res, *response), err
}

type ListTorrentsDataItem struct {
	Id       string        `json:"id"`                // Unique identifier of the torrent
	Filename string        `json:"filename"`          // Name of the torrent file
	Hash     string        `json:"hash"`              // SHA1 Hash of the torrent
	Bytes    int64         `json:"bytes"`             // Size of selected files only
	Host     string        `json:"host"`              // Host main domain
	Split    int           `json:"split"`             // Split size of links
	Progress float32       `json:"progress"`          // Possible values: 0 to 100
	Status   TorrentStatus `json:"status"`            // Current status of the torrent
	Added    string        `json:"added"`             // Date added (jsonDate format)
	Links    []string      `json:"links"`             // List of host URLs
	Ended    string        `json:"ended,omitempty"`   // Only present when finished, jsonDate format
	Speed    int64         `json:"speed,omitempty"`   // Only present in "downloading", "compressing", or "uploading" statuses
	Seeders  int           `json:"seeders,omitempty"` // Only present in "downloading" or "magnet_conversion" statuses
}

func (t ListTorrentsDataItem) GetAddedAt() time.Time {
	return toAddedAt(t.Added)
}

type ListTorrentsData = []ListTorrentsDataItem

type listTorrentsData struct {
	*ResponseError
	data ListTorrentsData
}

func (c *listTorrentsData) UnmarshalJSON(data []byte) error {
	var rerr ResponseError
	err := json.Unmarshal(data, &rerr)
	if err == nil {
		c.ResponseError = &rerr
		return nil
	}

	var items ListTorrentsData
	err = core.UnmarshalJSON(200, data, &items)
	if err == nil {
		c.data = items
		return nil
	}

	e := core.NewAPIError("failed to parse response")
	e.Cause = err
	return e
}

type ListTorrentsParams struct {
	Ctx
	Offset int    // Starting offset (must be within 0 and X-Total-Count HTTP header)
	Limit  int    // 	Entries returned per page / request (must be within 0 and 5000, default: 100)
	Page   int    //
	Filter string // "active", list active torrents only
}

func (c APIClient) ListTorrents(params *ListTorrentsParams) (APIResponse[ListTorrentsData], error) {
	form := &url.Values{}
	if params.Limit != 0 {
		form.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset != 0 {
		form.Add("offset", strconv.Itoa(params.Offset))
	}
	params.Form = form
	response := &listTorrentsData{}
	res, err := c.Request("GET", "/rest/1.0/torrents", params, response)
	return newAPIResponse(res, response.data), err
}

type StartTorrentDownloadData struct {
	*ResponseError
}

type StartTorrentDownloadParams struct {
	Ctx
	Id string
	// If only video file ids are present, each file will have separate download link.
	// If non-video file ids are present, the whole thing would be packaged/compressed into a single file and have one download link.
	// If not given, all files will be downloaded.
	FileIds []string
	IP      string
}

func (c APIClient) StartTorrentDownload(params *StartTorrentDownloadParams) (APIResponse[StartTorrentDownloadData], error) {
	fileIds := "all"
	if len(params.FileIds) > 0 {
		fileIds = strings.Join(params.FileIds, ",")
	}
	form := &url.Values{}
	form.Add("files", fileIds)
	if params.IP != "" {
		form.Add("ip", params.IP)
	}
	params.Form = form
	response := &StartTorrentDownloadData{}
	res, err := c.Request("POST", "/rest/1.0/torrents/selectFiles/"+params.Id, params, response)
	return newAPIResponse(res, *response), err
}

type UnrestrictLinkDataAlternative struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
	Download string `json:"download"`
	Type     string `json:"type"`
}

type UnrestrictLinkData struct {
	*ResponseError
	Id          string                          `json:"id"`
	Filename    string                          `json:"filename"`
	MimeType    string                          `json:"mimeType"`   // Mime Type of the file, guessed by the file extension
	Filesize    int                             `json:"filesize"`   // Filesize in bytes, 0 if unknown
	Link        string                          `json:"link"`       // Original link
	Host        string                          `json:"host"`       // Host main domain
	Chunks      int                             `json:"chunks"`     // Max chunks allowed
	CRC         int                             `json:"crc"`        // Disable/enable CRC check
	Download    string                          `json:"download"`   // Generated link
	Streamable  int                             `json:"streamable"` // Is the file streamable on website
	Type        string                          `json:"type"`       // Type of the file (e.g., quality)
	Alternative []UnrestrictLinkDataAlternative `json:"alternative"`
}

type UnrestrictLinkParams struct {
	Ctx
	Link     string // The original hoster link
	Password string // Password to unlock the file access hoster side
	Remote   int    // 0 or 1, use Remote traffic, dedicated servers and account sharing protections lifted
	IP       string
}

func (c APIClient) UnrestrictLink(params *UnrestrictLinkParams) (APIResponse[UnrestrictLinkData], error) {
	form := &url.Values{}
	form.Add("link", params.Link)
	if len(params.Password) > 0 {
		form.Add("password", params.Password)
	}
	if params.Remote != 0 {
		form.Add("remote", strconv.Itoa(params.Remote))
	}
	if params.IP != "" {
		form.Add("ip", params.IP)
	}
	params.Form = form
	response := &UnrestrictLinkData{}
	res, err := c.Request("POST", "/rest/1.0/unrestrict/link", params, response)
	return newAPIResponse(res, *response), err

}

type DeleteTorrentData struct {
	*ResponseError
}

type DeleteTorrentParams struct {
	Ctx
	Id string
}

func (c APIClient) DeleteTorrent(params *DeleteTorrentParams) (APIResponse[DeleteTorrentData], error) {
	response := &DeleteTorrentData{}
	res, err := c.Request("DELETE", "/rest/1.0/torrents/delete/"+params.Id, params, response)
	return newAPIResponse(res, *response), err

}
