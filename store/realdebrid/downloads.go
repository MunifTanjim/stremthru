package realdebrid

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type ListDownloadsDataItem struct {
	Id        string    `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mimeType"`  // Mime Type of the file, guessed by the file extension
	Filesize  int64     `json:"filesize"`  // bytes, 0 if unknown
	Link      string    `json:"link"`      // Original link
	Host      string    `json:"host"`      // Host main domain
	Chunks    int       `json:"chunks"`    // Max Chunks allowed
	Download  string    `json:"download"`  // Generated link
	Generated time.Time `json:"generated"` // jsonDate
	Type      string    `json:"type"`      // Type of the file (in general, its quality)
}

type ListDownloadsData []ListDownloadsDataItem

type listDownloadsData struct {
	*ResponseError
	data ListDownloadsData
}

func (c *listDownloadsData) UnmarshalJSON(data []byte) error {
	var rerr ResponseError
	err := json.Unmarshal(data, &rerr)
	if err == nil {
		c.ResponseError = &rerr
		return nil
	}

	var items ListDownloadsData
	err = core.UnmarshalJSON(200, data, &items)
	if err == nil {
		c.data = items
		return nil
	}

	e := core.NewAPIError("failed to parse response")
	e.Cause = err
	return e
}

type ListDownloadsParams struct {
	Ctx
	Offset int // Starting offset (must be within 0 and X-Total-Count HTTP header)
	Limit  int // Entries returned per page / request (must be within 0 and 5000, default: 100)
	Page   int //
}

func (c APIClient) ListDownloads(params *ListDownloadsParams) (APIResponse[ListDownloadsData], error) {
	query := &url.Values{}
	if params.Limit != 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset != 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	params.Query = query
	response := &listDownloadsData{}
	res, err := c.Request("GET", "/rest/1.0/downloads", params, response)
	return newAPIResponse(res, response.data), err
}
