package stremio

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type MetaPosterShape string

const (
	MetaPosterShapeSquare    MetaPosterShape = "square"    // 1:1
	MetaPosterShapePoster    MetaPosterShape = "poster"    // 1:0.675
	MetaPosterShapeLandscape MetaPosterShape = "landscape" // 1:1.77
)

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

	// undocumented
	MetaLinkCategoryIMDB      MetaLinkCategory = "imdb"
	MetaLinkCategoryShare     MetaLinkCategory = "share"
	MetaLinkCategoryGenres    MetaLinkCategory = "Genres"
	MetaLinkCategoryCast      MetaLinkCategory = "Cast"
	MetaLinkCategoryDirectors MetaLinkCategory = "Directors"
	MetaLinkCategoryWriters   MetaLinkCategory = "Writers"
)

type MetaLink struct {
	Name     string           `json:"name"`
	Category MetaLinkCategory `json:"category"`
	URL      string           `json:"url"`
}

type Number = util.JSONNumber

type ZeroIndexedInt int

func (zii ZeroIndexedInt) IsZero() bool {
	return zii == -1
}

func (zii ZeroIndexedInt) String() string {
	return strconv.Itoa(int(zii))
}

func (zii ZeroIndexedInt) Equal(i int) bool {
	return int(zii) == i
}

type MetaVideo struct {
	Id        string         `json:"id"`
	Title     string         `json:"title,omitempty"`
	Released  time.Time      `json:"released"`
	Thumbnail string         `json:"thumbnail,omitempty"`
	Streams   []Stream       `json:"streams,omitempty"`
	Available bool           `json:"available,omitempty"`
	Episode   ZeroIndexedInt `json:"episode,omitzero"`
	Season    ZeroIndexedInt `json:"season,omitzero"`
	Trailers  []Stream       `json:"trailers,omitempty"`
	Overview  string         `json:"overview,omitempty"`

	// deprecated / undocumented
	Name        string     `json:"name,omitempty"`
	MovieDBId   int        `json:"moviedb_id,omitempty"`
	TVDBId      int        `json:"tvdb_id,omitempty"`
	Rating      Number     `json:"rating,omitempty"`
	Description string     `json:"description,omitempty"`
	Number      int        `json:"number,omitempty"` // episode
	FirstAired  *time.Time `json:"firstAired,omitempty"`

	id_only bool `json:"-"`
}

type metaVideo MetaVideo

func (mv *MetaVideo) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		mv.Id = id
		mv.id_only = true
		return nil
	}
	_mv := metaVideo{}
	err := json.Unmarshal(data, &_mv)
	*mv = MetaVideo(_mv)
	return err
}

func (mv *MetaVideo) MarshalJSON() ([]byte, error) {
	if mv.id_only || (mv.Name == "" && mv.Description == "" && mv.Title == "" && mv.Overview == "") {
		return json.Marshal(mv.Id)
	}
	rsrc := metaVideo(*mv)
	return json.Marshal(&rsrc)
}

type MetaBehaviorHints struct {
	DefaultVideoId string `json:"defaultVideoId,omitempty"`

	// deprecated / undocumented
	HasScheduledVideos bool `json:"hasScheduledVideos,omitempty"`
}

// deprecated / undocumented
type MetaCreditsCastItem struct {
	Character   string `json:"character"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path,omitempty"`
	Id          int    `json:"id"`
}

// deprecated / undocumented
type MetaCreditsCrewItem struct {
	Department  string `json:"department"`
	Job         string `json:"job"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path,omitempty"`
	Id          int    `json:"id"`
}

// deprecated / undocumented
type MetaPopularities struct {
	Trakt      float32 `json:"trakt,omitempty"`
	MovieDB    float32 `json:"moviedb,omitempty"`
	Stremio    float32 `json:"stremio,omitempty"`
	StremioLib float32 `json:"stremio_lib,omitempty"`
}

type Meta struct {
	Id            string             `json:"id"`
	Type          ContentType        `json:"type"`
	Name          string             `json:"name"`
	Genres        []string           `json:"genres,omitempty"` // warning: this will soon be deprecated in favor of `links`
	Poster        string             `json:"poster,omitempty"`
	PosterShape   MetaPosterShape    `json:"posterShape,omitempty"`
	Background    string             `json:"background,omitempty"`
	Logo          string             `json:"logo,omitempty"`
	Description   string             `json:"description,omitempty"`
	ReleaseInfo   string             `json:"releaseInfo,omitempty"`
	Director      []string           `json:"director,omitempty"` // warning: this will soon be deprecated in favor of `links`
	Cast          []string           `json:"cast,omitempty"`     // warning: this will soon be deprecated in favor of `links`
	IMDBRating    string             `json:"imdbRating,omitempty"`
	Released      *time.Time         `json:"released,omitempty"`
	Trailers      []MetaTrailer      `json:"trailers,omitempty"` // warning: this will soon be deprecated in favor of `meta.trailers` being an array of `Stream`
	Links         []MetaLink         `json:"links,omitempty"`
	Videos        []MetaVideo        `json:"videos,omitempty"`
	Runtime       string             `json:"runtime,omitempty"`
	Language      string             `json:"language,omitempty"`
	Country       string             `json:"country,omitempty"`
	Awards        string             `json:"awards,omitempty"`
	Website       string             `json:"website,omitempty"`
	BehaviorHints *MetaBehaviorHints `json:"behaviorHints,omitempty"`

	// deprecated / undocumented
	CreditsCast    []MetaCreditsCastItem `json:"credits_cast,omitempty"`
	CreditsCrew    []MetaCreditsCrewItem `json:"credits_crew,omitempty"`
	DVDRelease     string                `json:"dvdRelease,omitempty"`
	Genre          []string              `json:"genre,omitempty"`
	IMDBId         string                `json:"imdb_id,omitempty"`
	MovieDBId      int                   `json:"moviedb_id,omitempty"`
	Popularity     float32               `json:"popularity,omitempty"`
	Popularities   *MetaPopularities     `json:"popularities,omitempty"`
	Slug           string                `json:"slug,omitempty"`
	Status         string                `json:"status,omitempty"` // 'Continuing' / 'Ended'
	TVDBId         Number                `json:"tvdb_id,omitempty"`
	TrailerStreams []Stream              `json:"trailerStreams,omitempty"`
	Writer         []string              `json:"writer,omitempty"`
	Year           string                `json:"year,omitempty"`
}

type MetaPreview struct {
	Id          string          `json:"id"`
	Type        ContentType     `json:"type"`
	Name        string          `json:"name"`
	Poster      string          `json:"poster"`
	PosterShape MetaPosterShape `json:"posterShape,omitempty"`

	Genres      []string      `json:"genres,omitempty"` // warning: this will soon be deprecated in favor of `links`
	IMDBRating  string        `json:"imdbRating,omitempty"`
	ReleaseInfo string        `json:"releaseInfo,omitempty"`
	Director    []string      `json:"director,omitempty"` // warning: this will soon be deprecated in favor of `links`
	Cast        []string      `json:"cast,omitempty"`     // warning: this will soon be deprecated in favor of `links`
	Links       []MetaLink    `json:"links,omitempty"`
	Description string        `json:"description,omitempty"`
	Trailers    []MetaTrailer `json:"trailers,omitempty"` // warning: this will soon be deprecated in favor of `meta.trailers` being an array of `Stream`

	Background    string             `json:"background,omitempty"`    // undocumented
	BehaviorHints *MetaBehaviorHints `json:"behaviorHints,omitempty"` // undocumented
}
