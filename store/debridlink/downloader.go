package debridlink

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DownloaderLink struct {
	Chunk        int    `json:"chunk"`
	Created      int64  `json:"created"`
	DownloadURL  string `json:"downloadUrl"`
	Expired      bool   `json:"expired"`
	Host         string `json:"host"`
	ID           string `json:"id"`
	IsProcessing bool   `json:"isProcessing"`
	Name         string `json:"name"`
	OtherLinks   []any  `json:"otherLinks"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
}

func (dl DownloaderLink) GetAddedAt() time.Time {
	return time.Unix(dl.Created, 0)
}

type ListDownloaderLinksData struct {
	Value      []DownloaderLink
	Pagination ResponsePagination
}

type ListDownloaderLinksParams struct {
	Ctx
	IDs     []string
	Page    int // start at 0
	PerPage int // min 20, max 100
	IP      string
}

func (c APIClient) ListDownloaderLinks(params *ListDownloaderLinksParams) (APIResponse[ListDownloaderLinksData], error) {
	query := &url.Values{}
	if len(params.IDs) > 0 {
		query.Add("ids", strings.Join(params.IDs, ","))
	}
	if params.Page != 0 {
		query.Add("page", strconv.Itoa(params.Page))
	}
	if params.PerPage != 0 {
		query.Add("perPage", strconv.Itoa(params.PerPage))
	}
	if params.IP != "" {
		query.Add("ip", params.IP)
	}
	params.Query = query

	response := &PaginatedResponse[DownloaderLink]{}
	res, err := c.Request("GET", "/v2/downloader/list", params, response)
	return newAPIResponse(res, ListDownloaderLinksData{
		Value:      response.Value,
		Pagination: response.Pagination,
	}), err
}
