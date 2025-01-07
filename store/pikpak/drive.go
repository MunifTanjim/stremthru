package pikpak

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type GetVIPInfoParams struct {
	Ctx
}

type GetVIPInfoDataVIPItem struct {
	Description string `json:"description"`
	Expire      string `json:"expire"`
	Status      string `json:"status"` // 'ok' / 'invalid'
	SurplusDay  int    `json:"surplus_day,omitempty"`
	Type        string `json:"type"` // 'regional'
}

type VIPType string

const (
	VIPTypePlatinum VIPType = "platinum"
	VIPTypeNovip    VIPType = "novip"
)

type GetVIPInfoData struct {
	Expire    string                  `json:"expire"`
	ExtType   string                  `json:"ext_type"`
	FeeRecord string                  `json:"fee_record"` // 'no_record'
	Kind      int                     `json:"kind"`       // 1 / 0
	Status    string                  `json:"status"`     // 'ok' / 'invalid'
	Type      VIPType                 `json:"type"`       // 'platinum' / 'novip'
	UserId    string                  `json:"user_id"`
	VIPItem   []GetVIPInfoDataVIPItem `json:"vipItem"`
}

type getVIPInfoData struct {
	ResponseContainer
	Data GetVIPInfoData `json:"data"`
}

func (c APIClient) GetVIPInfo(params *GetVIPInfoParams) (APIResponse[GetVIPInfoData], error) {
	response := &getVIPInfoData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, response.Data), err
	}
	err = c.withCaptchaToken(&params.Ctx, "", "")
	if err != nil {
		return newAPIResponse(nil, response.Data), err
	}

	res, err := c.DriveRequest("GET", "/vip/v1/vip/info", params, response)
	return newAPIResponse(res, response.Data), err
}

type FilePhase string

const (
	FilePhaseRunning  FilePhase = "PHASE_TYPE_RUNNING"
	FilePhaseError    FilePhase = "PHASE_TYPE_ERROR"
	FilePhaseComplete FilePhase = "PHASE_TYPE_COMPLETE"
	FilePhasePending  FilePhase = "PHASE_TYPE_PENDING"
)

type FileKind string

const (
	FileKindFile   FileKind = "drive#file"
	FileKindFolder FileKind = "drive#folder"
)

type FileMedia struct {
	MediaId   string `json:"media_id"`
	MediaName string `json:"media_name"`
	Video     struct {
		Height     int    `json:"height"`
		Width      int    `json:"width"`
		Duration   int    `json:"duration"`
		BitRate    int    `json:"bit_rate"`
		FrameRate  int    `json:"frame_rate"`
		VideoCodec string `json:"video_codec"`
		AudioCodec string `json:"audio_codec"`
		VideoType  string `json:"video_type"`
		HDRType    string `json:"hdr_type"` // 'HDR10'
	} `json:"video"`
	Link struct {
		URL    string `json:"url"`
		Token  string `json:"token"`
		Expire string `json:"expire"`
		Type   string `json:"type"`
	} `json:"link"`
	NeedMoreQuota  bool   `json:"need_more_quota"`
	VIPTypes       []any  `json:"vip_types"`
	RedirectLink   string `json:"redirect_link"`
	IconLink       string `json:"icon_link"`
	ExtIcon        string `json:"ext_icon"`
	IsDefault      bool   `json:"is_default"`
	Priority       int    `json:"priority"`
	IsOrigin       bool   `json:"is_origin"`
	ResolutionName string `json:"resolution_name"` // '480P'
	IsVisible      bool   `json:"is_visible"`
	Category       string `json:"category"` // 'category_origin' | 'category_transcode'
}

type File struct {
	Apps  []any `json:"apps"`
	Audit struct {
		Message string `json:"message"`
		Status  string `json:"status"` // 'STATUS_OK'
		Title   string `json:"title"`
	} `json:"audit"`
	CreatedTime       string      `json:"created_time"`
	DeleteTime        string      `json:"delete_time"`
	FileCategory      string      `json:"file_category"`  // 'ARCHIVE' | 'OTHER'
	FileExtension     string      `json:"file_extension"` // '.zip'
	FolderType        string      `json:"folder_type"`    // 'NORMAL' | 'DOWNLOAD'
	Hash              string      `json:"hash"`
	IconLink          string      `json:"icon_link"`
	Id                string      `json:"id"`
	Kind              FileKind    `json:"kind"`
	Links             struct{}    `json:"links"`
	MD5Checksum       string      `json:"md5_checksum"`
	Medias            []FileMedia `json:"medias"`
	MimeType          string      `json:"mime_type"` // 'application/zip'
	ModifiedTime      string      `json:"modified_time"`
	Name              string      `json:"name"`
	OriginalFileIndex int         `json:"original_file_index"`
	OriginalUrl       string      `json:"original_url"`
	Params            struct {
		GlobalFileKind  string `json:"global_file_kind,omitempty"` // '1'
		GlobalFileRoot  string `json:"global_file_root,omitempty"`
		GlobalFileToken string `json:"global_file_token,omitempty"`
		PlatformIcon    string `json:"platform_icon"`
		URL             string `json:"url"`
	} `json:"params"`
	ParentId         string    `json:"parent_id"`
	Phase            FilePhase `json:"phase"`
	ReferenceEvents  []any     `json:"reference_events"`
	Revision         string    `json:"revision"` // '0'
	Size             string    `json:"size"`
	SortName         string    `json:"sort_name"`
	Space            string    `json:"space"`
	SpellName        []any     `json:"spell_name"`
	Starred          bool      `json:"starred"`
	Tags             []any     `json:"tags"`
	ThumbnailLink    string    `json:"thumbnail_link"`
	Trashed          bool      `json:"trashed"`
	UserId           string    `json:"user_id"`
	UserModifiedTime string    `json:"user_modified_time"`
	WebContentLink   string    `json:"web_content_link"`
	Writable         bool      `json:"writable"`
}

type ListFilesData struct {
	ResponseContainer
	Files           []File `json:"files"` // 'drive#fileList'
	Kind            string `json:"kind"`
	NextPageToken   string `json:"next_page_token"`
	SyncTime        string `json:"sync_time"`
	Version         string `json:"version"`
	VersionOutdated bool   `json:"version_outdated"`
}

type ListFilesParams struct {
	Ctx
	ThumbnailSize string
	Limit         int
	ParentId      string
	WithAudit     bool
	PageToken     string
	Filters       map[string]map[string]any
}

func (c APIClient) ListFiles(params *ListFilesParams) (APIResponse[ListFilesData], error) {
	response := &ListFilesData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}
	err = c.withCaptchaToken(&params.Ctx, "", "")
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	if params.Query == nil {
		params.Query = &url.Values{}
	}

	if params.ThumbnailSize == "" {
		params.Query.Add("thumbnail_size", "SIZE_MEDIUM")
	} else {
		params.Query.Add("thumbnail_size", params.ThumbnailSize)
	}
	if params.Limit == 0 {
		params.Query.Add("limit", "100")
	} else {
		params.Query.Add("limit", string(params.Limit))
	}
	if params.ParentId != "" {
		params.Query.Add("parent_id", params.ParentId)
	}
	if params.WithAudit {
		params.Query.Add("with_audit", "true")
	}
	if params.PageToken != "" {
		params.Query.Add("page_token", params.PageToken)
	}
	if params.Filters != nil {
		filters, err := json.Marshal(params.Filters)
		if err == nil {
			params.Query.Add("filters", string(filters))
		}
	}

	res, err := c.DriveRequest("GET", "/drive/v1/files", params, response)
	return newAPIResponse(res, *response), err
}

type Task struct {
	Callback    string `json:"callback"`
	CreatedTime string `json:"created_time"`
	FileId      string `json:"file_id"`
	FileName    string `json:"file_name"`
	FileSize    string `json:"file_size"`
	IconLink    string `json:"icon_link"`
	Id          string `json:"id"`
	Kind        string `json:"kind"`
	Message     string `json:"message"`
	Name        string `json:"name"`
	Params      struct {
		Age         string `json:"age"`
		MimeType    string `json:"mime_type"`
		PredictType string `json:"predict_type"` // '1'
		URL         string `json:"url"`
	} `json:"params"`
	Phase             FilePhase `json:"phase"`
	Progress          int       `json:"progress"`
	ReferenceResource struct {
		Type  string `json:"@type"`
		Audit struct {
			Message string `json:"message"`
			Status  string `json:"status"` // 'STATUS_OK'
			Title   string `json:"title"`
		} `json:"audit"`
		Hash     string `json:"hash"`
		IconLink string `json:"icon_link"`
		Id       string `json:"id"`
		Kind     string `json:"kind"`
		Medias   []any  `json:"medias"`
		MimeType string `json:"mime_type"`
		Name     string `json:"name"`
		Params   struct {
			PlatformIcon string `json:"platform_icon"`
			URL          string `json:"url"`
		} `json:"params"`
		ParentId      string    `json:"parent_id"`
		Phase         FilePhase `json:"phase"`
		Size          string    `json:"size"`
		Space         string    `json:"space"`
		Starred       bool      `json:"starred"`
		Tags          []any     `json:"tags"`
		ThumbnailLink string    `json:"thumbnail_link"`
	}
	Space       string `json:"space"`
	StatusSize  int    `json:"status_size"`
	Statuses    []any  `json:"statuses"`
	ThirdTaskId string `json:"third_task_id"`
	Type        string `json:"type"` // 'offline'
	UpdatedTime string `json:"updated_time"`
	UserId      string `json:"user_id"`
}

type ListTasksData struct {
	ResponseContainer
	ExpiresIn     int    `json:"expires_in"`
	NextPageToken string `json:"next_page_token"`
	Tasks         []Task `json:"tasks"`
}

type ListTasksParams struct {
	Ctx
	With          string
	Type          string // 'offline'
	ThumbnailSize string
	Limit         int // default 10000
	Filters       map[string]map[string]any
	PageToken     string
}

func (c APIClient) ListTasks(params *ListTasksParams) (APIResponse[ListTasksData], error) {
	response := &ListTasksData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}
	err = c.withCaptchaToken(&params.Ctx, "", "")
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	if params.Query == nil {
		params.Query = &url.Values{}
	}

	if params.Type == "" {
		params.Query.Add("type", "offline")
	} else {
		params.Query.Add("type", params.Type)
	}
	if params.ThumbnailSize == "" {
		params.Query.Add("thumbnail_size", "SIZE_SMALL")
	} else {
		params.Query.Add("thumbnail_size", params.ThumbnailSize)
	}
	if params.Limit == 0 {
		params.Query.Add("limit", "10000")
	} else {
		params.Query.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.PageToken != "" {
		params.Query.Add("page_token", params.PageToken)
	}
	if params.Filters != nil {
		filters, err := json.Marshal(params.Filters)
		if err == nil {
			params.Query.Add("filters", string(filters))
		}
	}
	if params.With != "" {
		params.Query.Add("with", params.With)
	}

	res, err := c.DriveRequest("GET", "/drive/v1/tasks", params, response)
	return newAPIResponse(res, *response), err
}

type GetFileData struct {
	ResponseContainer
	File
}

type GetFileParams struct {
	Ctx
	FileId string
}

func (c APIClient) GetFile(params *GetFileParams) (APIResponse[GetFileData], error) {
	response := &GetFileData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}
	err = c.withCaptchaToken(&params.Ctx, "", "")
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	res, err := c.DriveRequest("GET", "/drive/v1/files/"+params.FileId, params, response)
	return newAPIResponse(res, *response), err
}

type AddFileDataURL struct {
	Kind string `json:"kind"` // 'upload#url'
}

type AddFileDataTask struct {
	Kind       string `json:"kind"` // 'drive#task'
	Id         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // 'offline'
	UserId     string `json:"user_id"`
	Statuses   []any  `json:"statuses"`
	StatusSize int    `json:"status_size"`
	Params     struct {
		PredictSpeed string `json:"predict_speed"`
		PredictType  string `json:"predict_type"` // '3'
	} `json:"params"`
	FileId      string    `json:"file_id"`
	FileName    string    `json:"file_name"`
	FileSize    string    `json:"file_size"`
	Message     string    `json:"message"`
	CreatedTime string    `json:"created_time"`
	UpdatedTime string    `json:"updated_time"`
	ThirdTaskId string    `json:"third_task_id"`
	Phase       FilePhase `json:"phase"`
	Progress    int       `json:"progress"`
	IconLink    string    `json:"icon_link"`
	Callback    string    `json:"callback"`
	Space       string    `json:"space"`
}

type AddFileData struct {
	ResponseContainer
	UploadType string          `json:"upload_type"` // 'UPLOAD_TYPE_URL'
	URL        AddFileDataURL  `json:"url"`
	Task       AddFileDataTask `json:"task"`
}

type AddFileParamsURL struct {
	URL string `json:"url"`
}

type AddFileParams struct {
	Ctx
	Kind       FileKind         `json:"kind"`
	URL        AddFileParamsURL `json:"url"`
	UploadType string           `json:"upload_type"`
	FolderType string           `json:"folder_type"`
	Name       string           `json:"name,omitempty"`
	ParentId   string           `json:"parent_id,omitempty"`
}

func (c APIClient) AddFile(params *AddFileParams) (APIResponse[AddFileData], error) {
	response := &AddFileData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}
	err = c.withCaptchaToken(&params.Ctx, "POST", "/drive/v1/files")
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	if params.Kind == "" {
		params.Kind = FileKindFile
	}
	if params.UploadType == "" {
		params.UploadType = "UPLOAD_TYPE_URL"
	}
	if params.ParentId == "" {
		params.FolderType = "DOWNLOAD"
	}

	params.JSON = params

	res, err := c.DriveRequest("POST", "/drive/v1/files", params, response)
	return newAPIResponse(res, *response), err
}

type TrashData struct {
	ResponseContainer
	TaskId string `json:"task_id"`
}

type TrashParams struct {
	Ctx
	Ids []string `json:"ids"`
}

func (c APIClient) Trash(params *TrashParams) (APIResponse[TrashData], error) {
	response := &TrashData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}
	err = c.withCaptchaToken(&params.Ctx, "", "")
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	params.JSON = params

	res, err := c.DriveRequest("POST", "/drive/v1/files:batchTrash", params, response)
	return newAPIResponse(res, *response), err
}
