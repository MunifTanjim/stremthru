package buddy

import (
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/store"
)

type CheckMagnetCacheParams struct {
	Ctx
	Store    store.StoreName
	Hashes   []string
	SId      string
	ClientIP string
}

type CheckMagnetCacheDataItemFile struct {
	Idx  int    `json:"i"`
	Name string `json:"n"`
	Size int    `json:"s"`
	SId  string `json:"sid"`
}

type CheckMagnetCacheDataItem struct {
	Hash   string                         `json:"hash"`
	Magnet string                         `json:"magnet"`
	Status store.MagnetStatus             `json:"status"`
	Files  []CheckMagnetCacheDataItemFile `json:"files"`
}

type CheckMagnetCacheData struct {
	Items      []CheckMagnetCacheDataItem `json:"items"`
	TotalItems int                        `json:"total_items"`
}

func (c APIClient) CheckMagnetCache(params *CheckMagnetCacheParams) (APIResponse[CheckMagnetCacheData], error) {
	params.Query = &url.Values{
		"hash": params.Hashes,
	}
	if params.SId != "" {
		params.Query.Set("sid", params.SId)
	}
	params.Headers = &http.Header{
		"X-StremThru-Store-Name": []string{string(params.Store)},
		"X-StremThru-Client-IP":  []string{params.ClientIP},
	}
	response := &Response[CheckMagnetCacheData]{}
	res, err := c.Request("GET", "/v0/store/magnet-cache/check", params, response)
	return newAPIResponse(res, response.Data), err
}

type TrackMagnetCacheData struct {
}

type TrackMagnetCacheParams struct {
	Ctx
	Store store.StoreName

	// single
	Hash      string             `json:"hash"`
	Files     []store.MagnetFile `json:"files"`
	CacheMiss bool               `json:"cache_miss"`

	// bulk
	FilesByHash map[string][]store.MagnetFile `json:"files_by_hash"`
}

func (c APIClient) TrackMagnetCache(params *TrackMagnetCacheParams) (APIResponse[TrackMagnetCacheData], error) {
	params.JSON = params
	params.Headers = &http.Header{
		"X-StremThru-Store-Name": []string{string(params.Store)},
	}
	response := &Response[TrackMagnetCacheData]{}
	res, err := c.Request("POST", "/v0/store/magnet-cache/track", params, response)
	return newAPIResponse(res, response.Data), err
}
