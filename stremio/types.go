package stremio

import "time"

type ResourceName string

const (
	ResourceNameCatalog      ResourceName = "catalog"
	ResourceNameMeta         ResourceName = "meta"
	ResourceNameStream       ResourceName = "stream"
	ResourceNameSubtitles    ResourceName = "subtitles"
	ResourceNameAddonCatalog ResourceName = "addon_catalog"
)

type ContentType string

const (
	ContentTypeMovie   ContentType = "movie"
	ContentTypeSeries  ContentType = "series"
	ContentTypeChannel ContentType = "channel"
	ContentTypeTV      ContentType = "tv"
)

type Resource struct {
	Name       ResourceName  `json:"name"`
	Types      []ContentType `json:"types"`
	IDPrefixes []string      `json:"idPrefixes,omitempty"`
}

type CatalogExtra struct {
	Name         string   `json:"name"`
	IsRequired   bool     `json:"isRequired,omitempty"`
	Options      []string `json:"options,omitempty"`
	OptionsLimit int      `json:"optionsLimit,omitempty"`
}

type Catalog struct {
	Type  string         `json:"type"`
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Extra []CatalogExtra `json:"extra,omitempty"`
}

type BehaviorHints struct {
	Adult                 bool `json:"adult,omitempty"`
	P2P                   bool `json:"p2p,omitempty"`
	Configurable          bool `json:"configurable,omitempty"`
	ConfigurationRequired bool `json:"configurationRequired,omitempty"`
}

type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	Resources  []Resource    `json:"resources"`
	Types      []ContentType `json:"types"`
	IDPrefixes []string      `json:"idPrefixes,omitempty"`

	Catalogs []Catalog `json:"catalogs"`

	Background    string         `json:"background,omitempty"`
	Logo          string         `json:"logo,omitempty"`
	ContactEmail  string         `json:"contactEmail,omitempty"`
	BehaviorHints *BehaviorHints `json:"behaviorHints,omitempty"`
}

type MetaTrailerType string

const (
	MetaTrailerTypeTrailer MetaTrailerType = "Trailer"
	MetaTrailerTypeClip    MetaTrailerType = "Clip"
)

type MetaTrailer struct {
	Source string          `json:"source"`
	Type   MetaTrailerType `json:"type"`
}

type MetaLinkCategory string

const (
	MetaLinkCategoryActor    MetaLinkCategory = "actor"
	MetaLinkCategoryDirector MetaLinkCategory = "director"
	MetaLinkCategoryWriter   MetaLinkCategory = "writer"
)

type MetaLink struct {
	Name     string           `json:"name"`
	Category MetaLinkCategory `json:"category"`
	URL      string           `json:"url"`
}

type Subtitle struct {
	Id   string `json:"id"`
	Url  string `json:"url"`
	Lang string `json:"lang"`
}

type StreamBehaviorHintsProxyHeaders struct {
	Request  map[string]string `json:"request,omitempty"`
	Response map[string]string `json:"response,omitempty"`
}

type StreamBehaviorHints struct {
	CountryWhitelist []string                         `json:"countryWhitelist,omitempty"`
	NotWebReady      bool                             `json:"notWebReady,omitempty"`
	BingeGroup       string                           `json:"bingeGroup,omitempty"`
	ProxyHeaders     *StreamBehaviorHintsProxyHeaders `json:"proxyHeaders,omitempty"`
	VideoHash        string                           `json:"videoHash,omitempty"`
	VideoSize        int64                            `json:"videoSize,omitempty"`
	Filename         string                           `json:"filename,omitempty"`
}

type Stream struct {
	URL         string `json:"url,omitempty"`
	YoutubeID   string `json:"ytId,omitempty"`
	InfoHash    string `json:"infoHash,omitempty"`
	FileIndex   int    `json:"fileIdx,omitempty"`
	ExternalURL string `json:"externalUrl,omitempty"`

	Name          string               `json:"name,omitempty"`
	Title         string               `json:"title,omitempty"` // deprecated, use `Description`
	Description   string               `json:"description,omitempty"`
	Subtitles     []Subtitle           `json:"subtitles,omitempty"`
	Sources       []string             `json:"sources,omitempty"`
	BehaviorHints *StreamBehaviorHints `json:"behaviorHints,omitempty"`
}

type MetaVideo struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	Released  time.Time `json:"released"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Streams   []Stream  `json:"streams,omitempty"`
	Available bool      `json:"available,omitempty"`
	Episode   int       `json:"episode,omitempty"`
	Season    int       `json:"season,omitempty"`
	Trailers  []Stream  `json:"trailers,omitempty"`
	Overview  string    `json:"overview,omitempty"`
}

type MetaBehaviorHints struct {
	DefaultVideoId string `json:"defaultVideoId,omitempty"`
}

type Meta struct {
	Id            string             `json:"id"`
	Type          ContentType        `json:"type"`
	Name          string             `json:"name"`
	Genres        []string           `json:"genres,omitempty"`
	Poster        string             `json:"poster,omitempty"`
	PosterShape   string             `json:"posterShape,omitempty"`
	Background    string             `json:"background,omitempty"`
	Logo          string             `json:"logo,omitempty"`
	Description   string             `json:"description,omitempty"`
	ReleaseInfo   string             `json:"releaseInfo,omitempty"`
	Director      []string           `json:"director,omitempty"`
	Cast          []string           `json:"cast,omitempty"`
	IMDBRating    string             `json:"imdbRating,omitempty"`
	Released      time.Time          `json:"released,omitempty"`
	Trailers      []MetaTrailer      `json:"trailers,omitempty"`
	Links         []MetaLink         `json:"links,omitempty"`
	Videos        []MetaVideo        `json:"videos,omitempty"`
	Runtime       string             `json:"runtime,omitempty"`
	Language      string             `json:"language,omitempty"`
	Country       string             `json:"country,omitempty"`
	Awards        string             `json:"awards,omitempty"`
	Website       string             `json:"website,omitempty"`
	BehaviorHints *MetaBehaviorHints `json:"behaviorHints,omitempty"`
}

type MetaPreview struct {
	Id          string        `json:"id"`
	Type        ContentType   `json:"type"`
	Name        string        `json:"name"`
	Poster      string        `json:"poster,omitempty"`
	PosterShape string        `json:"posterShape,omitempty"`
	Genres      []string      `json:"genres,omitempty"`
	IMDBRating  string        `json:"imdbRating,omitempty"`
	ReleaseInfo string        `json:"releaseInfo,omitempty"`
	Director    []string      `json:"director,omitempty"`
	Cast        []string      `json:"cast,omitempty"`
	Links       []MetaLink    `json:"links,omitempty"`
	Description string        `json:"description,omitempty"`
	Trailers    []MetaTrailer `json:"trailers,omitempty"`
}

type cacheControlResponse struct {
	CacheMaxAge     int `json:"cacheMaxAge,omitempty"`     // (in seconds) which sets the Cache-Control header to max-age=$cacheMaxAge
	StaleRevalidate int `json:"staleRevalidate,omitempty"` // (in seconds) which sets the Cache-Control header to stale-while-revalidate=$staleRevalidate
	StaleError      int `json:"staleError,omitempty"`      //  (in seconds) which sets the Cache-Control header to stale-if-error=$staleError
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
