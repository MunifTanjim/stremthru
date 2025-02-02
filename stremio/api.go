package stremio

type cacheControlResponse struct {
	CacheMaxAge     int `json:"cacheMaxAge,omitempty"`     // (in seconds) which sets the Cache-Control header to max-age=$cacheMaxAge
	StaleRevalidate int `json:"staleRevalidate,omitempty"` // (in seconds) which sets the Cache-Control header to stale-while-revalidate=$staleRevalidate
	StaleError      int `json:"staleError,omitempty"`      // (in seconds) which sets the Cache-Control header to stale-if-error=$staleError
}

type AddonCatalogHandlerResponse struct {
	cacheControlResponse
	Addons []Addon `json:"addons"`
}

type CatalogHandlerResponse struct {
	cacheControlResponse
	Metas []MetaPreview `json:"metas"`
}

type MetaHandlerResponse struct {
	cacheControlResponse
	Meta Meta `json:"meta"`
}

type StreamHandlerResponse struct {
	cacheControlResponse
	Streams []Stream `json:"streams"`
}

type SubtitlesHandlerResponse struct {
	cacheControlResponse
	Subtitles []Subtitle `json:"subtitles"`
}
