package seedr

import (
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type DeleteFolderData struct {
	ResponseContainer
}

type DeleteFolderParams struct {
	Ctx
	FolderId int64 `json:"folder_id"`
}

// NOTE: is buggy, returns error saying missing `delete_arr`
func (c *APIClient) DeleteFolder(params *DeleteFolderParams) (request.APIResponse[DeleteFolderData], error) {
	response := DeleteFolderData{}
	res, err := c.Request("DELETE", "/v0.1/p/fs/folder/"+strconv.FormatInt(params.FolderId, 10), params, &response)
	return request.NewAPIResponse(res, response), err
}

type BatchDeleteData struct {
	ResponseContainer
	Success bool `json:"success"`
}

type BatchDeleteParamsItem struct {
	Id   int64  `json:"id"`
	Type string `json:"type"` // "file" / "folder" / "torrent"
}

type BatchDeleteParams struct {
	Ctx
	Items []BatchDeleteParamsItem `json:"delete_arr"`
}

func (c *APIClient) BatchDelete(params *BatchDeleteParams) (request.APIResponse[BatchDeleteData], error) {
	params.JSON = params

	response := BatchDeleteData{}
	res, err := c.Request("POST", "/v0.1/p/fs/batch/delete", params, &response)
	return request.NewAPIResponse(res, response), err
}
