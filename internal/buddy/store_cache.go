package buddy

import (
	"net/http"
	"net/url"

	"github.com/MunifTanjim/stremthru/store"
)

type CheckMagnetCacheParams struct {
	Ctx
	Store  store.StoreName
	Hashes []string
}

func (c APIClient) CheckMagnetCache(params *CheckMagnetCacheParams) (APIResponse[store.CheckMagnetData], error) {
	params.Form = &url.Values{
		"hash": params.Hashes,
	}
	params.Headers = &http.Header{
		"X-StremThru-Store-Name": []string{string(params.Store)},
	}
	response := &Response[store.CheckMagnetData]{}
	res, err := c.Request("GET", "/v0/store/magnet-cache/check", params, response)
	return newAPIResponse(res, response.Data), err
}

type TrackMagnetCacheData struct {
}

type TrackMagnetCacheParams struct {
	Ctx
	Store     store.StoreName
	Hash      string             `json:"hash"`
	Files     []store.MagnetFile `json:"files"`
	CacheMiss bool               `json:"cache_miss"`
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
