package premiumize

import (
	"net/url"
	"time"
)

type VirusScan string

const (
	VirusScanError    VirusScan = "error"
	VirusScanInfected VirusScan = "infected"
	VirusScanOk       VirusScan = "ok"
)

type ListFolderDataBreadcrumbItem struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	ParentId string `json:"parent_id"`
}

type FolderItemType string

const (
	FolderItemTypeFolder FolderItemType = "folder"
	FolderItemTypeFile   FolderItemType = "file"
)

type ListFolderDataContentItem struct {
	Id              string          `json:"id"`
	Name            string          `json:"name"`
	Type            FolderItemType  `json:"type"`
	Size            int64           `json:"size"`
	CreatedAt       int64           `json:"created_at"`
	MimeType        string          `json:"mime_type"`
	TranscodeStatus TranscodeStatus `json:"transcode_status"`
	Link            string          `json:"link"`
	StreamLink      string          `json:"stream_link"`
	VirusScan       VirusScan       `json:"virus_scan"`
}

func (c ListFolderDataContentItem) GetAddedAt() time.Time {
	return time.Unix(c.CreatedAt, 0).UTC()
}

type ListFoldersData struct {
	Content     []ListFolderDataContentItem    `json:"content"`
	Breadcrumbs []ListFolderDataBreadcrumbItem `json:"breadcrumbs"`
	Name        string                         `json:"name"`
	ParentId    string                         `json:"parent_id"`
	FolderId    string                         `json:"folder_id"`
}
type listFoldersData struct {
	ResponseContainer
	ListFoldersData
}

type ListFoldersParams struct {
	Ctx
	Id                 string
	IncludeBreadcrumbs bool
}

func (c APIClient) ListFolders(params *ListFoldersParams) (APIResponse[ListFoldersData], error) {
	form := &url.Values{}
	if params.Id != "" {
		form.Add("id", params.Id)
	}
	if params.IncludeBreadcrumbs {
		form.Add("includebreadcrumbs", "true")
	}
	params.Form = form

	response := &listFoldersData{}
	res, err := c.Request("GET", "/folder/list", params, response)
	return newAPIResponse(res, response.ListFoldersData), err
}

type CreateFolderData struct {
	Id string `json:"id"`
}

type createFolderData struct {
	ResponseContainer
	CreateFolderData
}

type CreateFolderParams struct {
	Ctx
	Name     string
	ParentId string
}

func (c APIClient) CreateFolder(params *CreateFolderParams) (APIResponse[CreateFolderData], error) {
	form := &url.Values{}
	form.Add("name", params.Name)
	if params.ParentId != "" {
		form.Add("parent_id", params.ParentId)
	}
	params.Form = form

	response := &createFolderData{}
	res, err := c.Request("POST", "/folder/create", params, response)
	return newAPIResponse(res, response.CreateFolderData), err
}

type SearchFoldersData struct {
	Content []ListFolderDataContentItem `json:"content"`
}

type searchFoldersData struct {
	ResponseContainer
	SearchFoldersData
}

type SearchFoldersParams struct {
	Ctx
	Query string
}

func (c APIClient) SearchFolders(params *SearchFoldersParams) (APIResponse[SearchFoldersData], error) {
	form := &url.Values{}
	form.Add("q", params.Query)
	params.Form = form

	response := &searchFoldersData{}
	res, err := c.Request("GET", "/folder/search", params, response)
	return newAPIResponse(res, response.SearchFoldersData), err
}

type DeleteFolderData struct {
}

type deleteFolderData struct {
	ResponseContainer
	DeleteFolderData
}

type DeleteFolderParams struct {
	Ctx
	Id string
}

func (c APIClient) DeleteFolder(params *DeleteFolderParams) (APIResponse[DeleteFolderData], error) {
	form := &url.Values{}
	form.Add("id", params.Id)
	params.Form = form

	response := &deleteFolderData{}
	res, err := c.Request("POST", "/folder/delete", params, response)
	return newAPIResponse(res, response.DeleteFolderData), err
}
