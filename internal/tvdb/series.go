package tvdb

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/request"
)

type SeriesStatusId int

const (
	SeriesStatusContinuing SeriesStatusId = iota + 1
	SeriesStatusEnded
	SeriesStatusUpcoming
)

type SeriesStatus struct {
	Id          SeriesStatusId `json:"id"`
	Name        string         `json:"name"`       // Continuing | Ended | Upcoming
	RecordType  string         `json:"recordType"` // series
	KeepUpdated bool           `json:"keepUpdated"`
}

type Series struct {
	Id                   int          `json:"id"`
	Name                 string       `json:"name"`
	Slug                 string       `json:"slug"`
	Image                string       `json:"image"`
	NameTranslations     []string     `json:"nameTranslations"`
	OverviewTranslations []string     `json:"overviewTranslations"`
	Aliases              []Alias      `json:"aliases"`
	FirstAired           string       `json:"firstAired"`
	LastAired            string       `json:"lastAired"`
	NextAired            string       `json:"nextAired"`
	Score                int64        `json:"score"`
	Status               SeriesStatus `json:"status"`
	OriginalCountry      string       `json:"originalCountry"`
	OriginalLanguage     string       `json:"originalLanguage"`
	DefaultSeasonType    int          `json:"defaultSeasonType"`
	IsOrderRandomized    bool         `json:"isOrderRandomized"`
	LastUpdated          string       `json:"lastUpdated"`
	AverageRuntime       int          `json:"averageRuntime"`
	Overview             string       `json:"overview"`
	Year                 string       `json:"year"`
}

type SeasonType struct {
	Id            int    `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"` // official / dvd / absolute
	AlternateName string `json:"alternateName"`
}

type Season struct {
	Id                   int        `json:"id"`
	SeriesId             int        `json:"seriesId"`
	Type                 SeasonType `json:"type"`
	Number               int        `json:"number"`
	NameTranslations     []string   `json:"nameTranslations"`
	OverviewTranslations []string   `json:"overviewTranslations"`
	Image                string     `json:"image"`
	ImageType            int        `json:"imageType"`
	Companies            Companies  `json:"companies"`
	LastUpdated          string     `json:"lastUpdated"`
}

type Episode struct {
	Id                   int         `json:"id"`
	SeriesId             int         `json:"seriesId"`
	Name                 string      `json:"name"`
	Aired                string      `json:"aired"` // YYYY-MM-DD
	Runtime              int         `json:"runtime"`
	NameTranslations     []any       `json:"nameTranslations"`
	Overview             string      `json:"overview"`
	OverviewTranslations []any       `json:"overviewTranslations"`
	Image                string      `json:"image"` // /banners/episodes/<seriesid>/<...>.jpg
	ImageType            ArtworkType `json:"imageType"`
	IsMovie              int         `json:"isMovie"`
	Seasons              []any       `json:"seasons,omitempty"`
	Number               int         `json:"number"`
	AbsoluteNumber       int         `json:"absoluteNumber"`
	SeasonNumber         int         `json:"seasonNumber"`
	LastUpdated          string      `json:"lastUpdated"`
	FinaleType           string      `json:"finaleType,omitempty"` // series / season
	Year                 string      `json:"year"`
}

type Translations struct {
	NameTranslations     []NameTranslation     `json:"nameTranslations"`
	OverviewTranslations []OverviewTranslation `json:"overviewTranslations"`
	Aliases              []string              `json:"aliases"`
}

func (t Translations) GetOverview() string {
	overview := ""
	for i := range t.OverviewTranslations {
		o := t.OverviewTranslations[i]
		if o.IsPrimary {
			return o.Overview
		} else if o.Language == "eng" {
			overview = o.Overview
		}
	}
	return overview
}

type AirDays struct {
	Sunday    bool `json:"sunday"`
	Monday    bool `json:"monday"`
	Tuesday   bool `json:"tuesday"`
	Wednesday bool `json:"wednesday"`
	Thursday  bool `json:"thursday"`
	Friday    bool `json:"friday"`
	Saturday  bool `json:"saturday"`
}

type ExtendedSeries struct {
	Series
	Episodes        []Episode         `json:"episodes"`
	Artworks        []ExtendedArtwork `json:"artworks"`
	Companies       []Company         `json:"companies"`
	OriginalNetwork Company           `json:"originalNetwork"`
	LatestNetwork   Company           `json:"latestNetwork"`
	Genres          []Genre           `json:"genres"`
	Trailers        []Trailer         `json:"trailers"`
	Lists           []List            `json:"lists"`
	RemoteIds       []RemoteId        `json:"remoteIds"`
	Characters      []Character       `json:"characters"`
	AirsDays        AirDays           `json:"airsDays"`
	AirsTime        string            `json:"airsTime"`
	Seasons         []Season          `json:"seasons"`
	Translations    Translations      `json:"translations"`
	Tags            []TagOption       `json:"tags"`
	ContentRatings  []ContentRating   `json:"contentRatings"`
	SeasonTypes     []SeasonType      `json:"seasonTypes"`
}

func (s *ExtendedSeries) GetPoster() string {
	for i := range s.Artworks {
		artwork := &s.Artworks[i]
		if artwork.Type == ArtworkTypeSeriesPoster {
			return artwork.Image
		}
	}
	return ""
}

func (s *ExtendedSeries) GetBackground() string {
	for i := range s.Artworks {
		artwork := &s.Artworks[i]
		if artwork.Type == ArtworkTypeSeriesBackground {
			return artwork.Image
		}
	}
	return ""
}

func (s *ExtendedSeries) GetClearLogo() string {
	for i := range s.Artworks {
		artwork := &s.Artworks[i]
		if artwork.Type == ArtworkTypeSeriesClearLogo {
			return artwork.Image
		}
	}
	return ""
}

func (s *ExtendedSeries) GetTrailer() string {
	trailer := ""
	for i := range s.Trailers {
		t := &s.Trailers[i]
		if t.Language == "eng" {
			return t.URL
		} else {
			trailer = t.URL
		}
	}
	return trailer
}

func (s *ExtendedSeries) GetIdMap() *meta.IdMap {
	idMap := meta.IdMap{Type: meta.IdTypeShow}
	idMap.TVDB = strconv.Itoa(s.Id)
	for i := range s.RemoteIds {
		rid := &s.RemoteIds[i]
		switch rid.Type {
		case SourceTypeIMDB:
			idMap.IMDB = rid.Id
		case SourceTypeTMDBTV:
			idMap.TMDB = rid.Id
		}
	}
	return &idMap
}

type FetchSeriesParams struct {
	Ctx
	Id int
}

func (c APIClient) FetchSeries(params *FetchSeriesParams) (request.APIResponse[ExtendedSeries], error) {
	params.Query = &url.Values{
		"meta": []string{"episodes,translations"},
	}
	response := Response[ExtendedSeries]{}
	res, err := c.Request("GET", "/series/"+strconv.Itoa(params.Id)+"/extended", params, &response)
	return request.NewAPIResponse(res, response.Data), err
}
