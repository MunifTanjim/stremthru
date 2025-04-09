package alldebrid

import (
	"encoding/json"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/store"
)

type MagnetError struct {
	Code    MagnetErrorCode `json:"code"`
	Message string          `json:"message"`
}

func (e *MagnetError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseMagnetFile struct {
	Children *[]ResponseMagnetFile `json:"e"`
	Link     string                `json:"l"`
	Name     string                `json:"n"`
	Size     int64                 `json:"s"`
}

type MagnetFile struct {
	Idx  int
	Link string
	Name string
	Path string
	Size int64
	Type store.MagnetFileType
}

func getFlatFiles(files *[]ResponseMagnetFile, result []MagnetFile, parent *MagnetFile, idx int) []MagnetFile {
	if result == nil {
		result = []MagnetFile{}
	}

	for _, f := range *files {
		file := &MagnetFile{
			Idx:  idx,
			Name: f.Name,
			Path: "/" + f.Name,
			Size: f.Size,
			Type: store.MagnetFileTypeFile,
			Link: f.Link,
		}

		if parent != nil {
			file.Path = path.Join(parent.Path, file.Name)
		}

		if f.Children == nil {
			result = append(result, *file)
			idx++
		} else {
			file.Type = store.MagnetFileTypeFolder
			result = append(result, *file)
			idx++
			result = getFlatFiles(f.Children, result, file, idx)
		}
	}

	return result
}

type MagnetFilesContainer struct {
	Files *[]ResponseMagnetFile `json:"files,omitempty"`
}

func (m MagnetFilesContainer) GetFiles() []MagnetFile {
	return getFlatFiles(m.Files, nil, nil, 0)
}

type GetMagnetInstantDataMagnet struct {
	MagnetFilesContainer
	Error   *MagnetError `json:"error,omitempty"`
	Hash    string       `json:"hash"`
	Instant bool         `json:"instant"`
	Magnet  string       `json:"magnet"`
}

type GetMagnetInstantData struct {
	Magnets []GetMagnetInstantDataMagnet `json:"magnets"`
}

type GetMagnetInstantParams struct {
	Ctx
	Magnets []string
}

// Deprecated: AllDebrid removed the endpoint
func (c APIClient) GetMagnetInstant(params *GetMagnetInstantParams) (APIResponse[[]GetMagnetInstantDataMagnet], error) {
	params.Form = &url.Values{
		"magnets[]": params.Magnets,
	}

	response := &Response[GetMagnetInstantData]{}
	res, err := c.Request("POST", "/v4/magnet/instant", params, response)
	return newAPIResponse(res, response.Data.Magnets), err
}

type UploadMagnetDataMagnet struct {
	Error            *MagnetError `json:"error,omitempty"`
	FilenameOriginal string       `json:"filename_original"`
	Hash             string       `json:"hash"`
	Id               int          `json:"id"`
	Magnet           string       `json:"magnet"`
	Name             string       `json:"name"`
	Ready            bool         `json:"ready"`
	Size             int64        `json:"size"`
}

type UploadMagnetData struct {
	Magnets []UploadMagnetDataMagnet `json:"magnets"`
}

type UploadMagnetParams struct {
	Ctx
	Magnets []string
}

func (c APIClient) UploadMagnet(params *UploadMagnetParams) (APIResponse[[]UploadMagnetDataMagnet], error) {
	params.Form = &url.Values{
		"magnets[]": params.Magnets,
	}

	response := &Response[UploadMagnetData]{}
	res, err := c.Request("POST", "/v4/magnet/upload", params, response)
	return newAPIResponse(res, response.Data.Magnets), err
}

type MagnetStatusCode int

const (
	MagnetStatusCodeQueued                   MagnetStatusCode = iota // In Queue
	MagnetStatusCodeDownloading                                      // Downloading
	MagnetStatusCodeProcessing                                       // Compressing / Moving
	MagnetStatusCodeUploading                                        // Uploading
	MagnetStatusCodeReady                                            // Ready
	MagnetStatusCodeUploadFailed                                     // Upload fail
	MagnetStatusCodeErrorUnpackFailed                                // Internal error on unpacking
	MagnetStatusCodeErrorDownloadTimedOut                            // Not downloaded in 20 min
	MagnetStatusCodeErrorFileTooBig                                  // File too big
	MagnetStatusCodeErrorUnknown                                     // Internal error
	MagnetStatusCodeErrorDownloadTookTooLong                         // Download took more than 72h
	MagnetStatusCodeErrorDeletedUpstream                             // Deleted on the hoster website
)

// get magnet files
type GetMagnetFilesDataMagnet struct {
	MagnetFilesContainer
	Id int `json:"id"`
}

type GetMagnetFilesData struct {
	Magnets []GetMagnetFilesDataMagnet `json:"magnets"`
}

type GetMagnetFilesParams struct {
	Ctx
	Ids []int
}

func (c APIClient) GetMagnetFiles(params *GetMagnetFilesParams) (APIResponse[[]GetMagnetFilesDataMagnet], error) {
	form := &url.Values{}
	for _, id := range params.Ids {
		form.Add("id[]", strconv.Itoa(id))
	}
	params.Form = form

	response := &Response[GetMagnetFilesData]{}
	res, err := c.Request("GET", "/v4/magnet/files", params, response)
	return newAPIResponse(res, response.Data.Magnets), err
}

type GetMagnetStatusDataMagnet struct {
	MagnetFilesContainer
	Id             int              `json:"id"`
	Filename       string           `json:"filename"`
	Size           int64            `json:"size"`
	Hash           string           `json:"hash"`
	Type           string           `json:"type"`
	Version        int              `json:"version"`
	Status         string           `json:"status"`
	StatusCode     MagnetStatusCode `json:"statusCode"`
	Downloaded     int              `json:"downloaded"`
	Uploaded       int              `json:"uploaded"`
	Seeders        int              `json:"seeders"`
	DownloadSpeed  int              `json:"downloadSpeed"`
	UploadSpeed    int              `json:"uploadSpeed"`
	UploadDate     int64            `json:"uploadDate"`
	CompletionDate int64            `json:"completionDate"`
}

func (m GetMagnetStatusDataMagnet) GetAddedAt() time.Time {
	unixSeconds := m.CompletionDate
	if unixSeconds == 0 {
		unixSeconds = m.UploadDate
	}
	return time.Unix(unixSeconds, 0).UTC()
}

type GetMagnetStatusData struct {
	Magnet GetMagnetStatusDataMagnet `json:"magnets"`
}

type GetMagnetStatusParams struct {
	Ctx
	Id int
}

func (c APIClient) GetMagnetStatus(params *GetMagnetStatusParams) (APIResponse[GetMagnetStatusDataMagnet], error) {
	form := &url.Values{}
	form.Add("id", strconv.Itoa(params.Id))
	params.Form = form

	response := &Response[GetMagnetStatusData]{}
	res, err := c.Request("GET", "/v4.1/magnet/status", params, response)
	return newAPIResponse(res, response.Data.Magnet), err
}

type GetAllMagnetStatusData struct {
	Magnets  []GetMagnetStatusDataMagnet `json:"magnets"`
	Counter  int                         `json:"counter"`
	FullSync bool                        `json:"fullsync"`
}

type GetAllMagnetStatusParams struct {
	Ctx
	Session int
	Counter int
}

func (c APIClient) GetAllMagnetStatus(params *GetAllMagnetStatusParams) (APIResponse[GetAllMagnetStatusData], error) {
	form := &url.Values{}
	if params.Session != 0 {
		form.Add("session", strconv.Itoa(params.Session))
		form.Add("counter", strconv.Itoa(params.Counter))
	}
	params.Form = form

	response := &Response[GetAllMagnetStatusData]{}
	res, err := c.Request("GET", "/v4.1/magnet/status", params, response)
	return newAPIResponse(res, response.Data), err
}

type DeleteMagnetData struct {
	Message string `json:"message"`
}

type DeleteMagnetParams struct {
	Ctx
	Id int
}

func (c APIClient) DeleteMagnet(params *DeleteMagnetParams) (APIResponse[DeleteMagnetData], error) {
	form := &url.Values{}
	form.Add("id", strconv.Itoa(params.Id))
	params.Form = form

	response := &Response[DeleteMagnetData]{}
	res, err := c.Request("GET", "/v4/magnet/delete", params, response)
	return newAPIResponse(res, response.Data), err
}

type RestartMagnetData struct {
	Message string `json:"message"`
}

type RestartMagnetParams struct {
	Ctx
	Id int
}

func (c APIClient) RestartMagnet(params *RestartMagnetParams) (APIResponse[RestartMagnetData], error) {
	form := &url.Values{}
	form.Add("id", strconv.Itoa(params.Id))
	params.Form = form

	response := &Response[RestartMagnetData]{}
	res, err := c.Request("GET", "/v4/magnet/restart", params, response)
	return newAPIResponse(res, response.Data), err
}

type RestartMagnetsDataMagnet struct {
	Magnet  int          `json:"magnet"`
	Message string       `json:"message"`
	Error   *MagnetError `json:"error,omitempty"`
}

type RestartMagnetsData struct {
	Magnets []RestartMagnetsDataMagnet `json:"magnets"`
}

type RestartMagnetsParams struct {
	Ctx
	Ids []int
}

func (c APIClient) RestartMagnets(params *RestartMagnetsParams) (APIResponse[[]RestartMagnetsDataMagnet], error) {
	form := &url.Values{}
	for _, id := range params.Ids {
		form.Add("ids[]", strconv.Itoa(id))
	}
	params.Form = form

	response := &Response[RestartMagnetsData]{}
	res, err := c.Request("GET", "/v4/magnet/restart", params, response)
	return newAPIResponse(res, response.Data.Magnets), err
}
