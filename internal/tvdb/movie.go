package tvdb

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type MovieStatusId int

const (
	MovieStatusAnnounced MovieStatusId = iota + 1
	MovieStatusPreProduction
	MovieStatusFilmingOrPostProduction
	MovieStatusCompleted
	MovieStatusReleased
)

type MovieStatus struct {
	Id          MovieStatusId `json:"id"`
	Name        string        `json:"name"`       // Announced | Pre-Production | Filming / Post-Production | Completed | Released
	RecordType  string        `json:"recordType"` // movie
	KeepUpdated bool          `json:"keepUpdated"`
}

type Movie struct {
	Id                   int         `json:"id"`
	Name                 string      `json:"name"`
	Slug                 string      `json:"slug"`
	Image                string      `json:"image"`
	NameTranslations     []string    `json:"nameTranslations"`
	OverviewTranslations []string    `json:"overviewTranslations"`
	Aliases              []Alias     `json:"aliases"`
	Score                int64       `json:"score"`
	Runtime              int         `json:"runtime"`
	Status               MovieStatus `json:"status"`
	LastUpdated          string      `json:"lastUpdated"`
	Year                 string      `json:"year"`
}

type Trailer struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Language string `json:"language"`
	Runtime  int    `json:"runtime"`
}

type Release struct {
	Country string `json:"country"` // global
	Date    string `json:"date"`
	Detail  any    `json:"detail"`
}

type Studio struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ParentStudio any    `json:"parentStudio"`
}

type NameTranslation struct {
	Name      string `json:"name"`
	Language  string `json:"language"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
	Tagline   string `json:"tagline,omitempty"`
}

type OverviewTranslation struct {
	Overview  string `json:"overview"`
	Language  string `json:"language"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
	Tagline   string `json:"tagline,omitempty"`
}

type ExtendedMovie struct {
	Movie
	Trailers            []Trailer           `json:"trailers"`
	Genres              []Genre             `json:"genres"`
	Releases            []Release           `json:"releases"`
	Artworks            []Artwork           `json:"artworks"`
	RemoteIds           []RemoteId          `json:"remoteIds"`
	Characters          Characters          `json:"characters"`
	Budget              util.JSONNumber     `json:"budget,omitempty"`
	BoxOffice           util.JSONNumber     `json:"boxOffice,omitempty"`
	BoxOfficeUS         util.JSONNumber     `json:"boxOfficeUS,omitempty"`
	OriginalCountry     CountryId           `json:"originalCountry"`
	OriginalLanguage    string              `json:"originalLanguage"`
	AudioLanguages      []string            `json:"audioLanguages"`
	SubtitleLanguages   []string            `json:"subtitleLanguages"`
	Studios             []Studio            `json:"studios"`
	Awards              Awards              `json:"awards"`
	TagOptions          []TagOption         `json:"tagOptions"`
	Lists               []List              `json:"lists"`
	Translations        Translations        `json:"translations"`
	Companies           Companies           `json:"companies"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	Inspirations        []any               `json:"inspirations"`
	SpokenLanguages     []string            `json:"spoken_languages"`
	FirstRelease        Release             `json:"first_release"`
}

func (m *ExtendedMovie) GetPoster() string {
	for i := range m.Artworks {
		artwork := &m.Artworks[i]
		if artwork.Type == ArtworkTypeMoviePoster {
			return artwork.Image
		}
	}
	return ""
}

func (m *ExtendedMovie) GetBackground() string {
	for i := range m.Artworks {
		artwork := &m.Artworks[i]
		if artwork.Type == ArtworkTypeMovieBackground {
			return artwork.Image
		}
	}
	return ""
}

func (m *ExtendedMovie) GetClearLogo() string {
	for i := range m.Artworks {
		artwork := &m.Artworks[i]
		if artwork.Type == ArtworkTypeMovieClearLogo {
			return artwork.Image
		}
	}
	return ""
}

func (m *ExtendedMovie) GetTrailer() string {
	trailer := ""
	for i := range m.Trailers {
		t := &m.Trailers[i]
		if t.Language == "eng" {
			return t.URL
		} else {
			trailer = t.URL
		}
	}
	return trailer
}

func (m *ExtendedMovie) GetIdMap() *meta.IdMap {
	idMap := meta.IdMap{Type: meta.IdTypeMovie}
	idMap.TVDB = strconv.Itoa(m.Id)
	for i := range m.RemoteIds {
		rid := &m.RemoteIds[i]
		switch rid.Type {
		case SourceTypeIMDB:
			idMap.IMDB = rid.Id
		case SourceTypeTMDB:
			idMap.TMDB = rid.Id
		}
	}
	return &idMap
}

type FetchMovieParams struct {
	Ctx
	Id int
}

func (c APIClient) FetchMovie(params *FetchMovieParams) (request.APIResponse[ExtendedMovie], error) {
	params.Query = &url.Values{
		"meta": []string{"translations"},
	}
	response := Response[ExtendedMovie]{}
	res, err := c.Request("GET", "/movies/"+strconv.Itoa(params.Id)+"/extended", params, &response)
	return request.NewAPIResponse(res, response.Data), err
}
