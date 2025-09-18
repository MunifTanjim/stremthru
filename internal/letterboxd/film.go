package letterboxd

import "github.com/MunifTanjim/stremthru/internal/meta"

type FilmSummary struct {
	Id                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	OriginalName         string                   `json:"originalName,omitempty"` // FIRST PARTY
	SortingName          string                   `json:"sortingName"`
	AlternativeNames     []string                 `json:"alternativeNames,omitempty"` // FIRST PARTY
	ReleaseYear          int                      `json:"releaseYear"`
	RunTime              int                      `json:"runTime,omitempty"`
	Rating               float32                  `json:"rating,omitempty"`
	Directors            []ContributorSummary     `json:"directors"`
	Poster               *Image                   `json:"poster,omitempty"`
	AdultPoster          *Image                   `json:"adultPoster,omitempty"`
	Top250Position       int32                    `json:"top250Position,omitempty"`
	Adult                bool                     `json:"adult"`
	ReviewsHidden        bool                     `json:"reviewsHidden"`
	PosterCustomizable   bool                     `json:"posterCustomizable"`
	BackdropCustomizable bool                     `json:"backdropCustomizable"`
	FilmCollectionId     string                   `json:"filmCollectionId,omitempty"`
	Links                []Link                   `json:"links"`
	Relationships        []MemberFilmRelationship `json:"relationships,omitempty"`
	Genres               []Genre                  `json:"genres"`
	PosterPickerURL      string                   `json:"posterPickerUrl,omitempty"`   // FIRST PARTY
	BackdropPickerURL    string                   `json:"backdropPickerUrl,omitempty"` // FIRST PARTY
}

func (fs FilmSummary) GenreIds() []string {
	ids := make([]string, len(fs.Genres))
	for i := range fs.Genres {
		ids[i] = fs.Genres[i].Id
	}
	return ids
}

func (fs FilmSummary) GetPoster() string {
	var sizes []ImageSize
	if fs.Adult && fs.AdultPoster != nil {
		sizes = fs.AdultPoster.Sizes
	} else if fs.Poster != nil {
		sizes = fs.Poster.Sizes
	}
	for i := range sizes {
		size := &sizes[i]
		if size.Width >= 300 {
			return size.URL
		}
	}
	return ""
}

func (fs FilmSummary) GetIdMap() *meta.IdMap {
	idMap := meta.IdMap{Type: meta.IdTypeMovie}
	for i := range fs.Links {
		link := &fs.Links[i]
		switch link.Type {
		case LinkTypeLetterboxd:
			idMap.Letterboxd = link.Id
		case LinkTypeIMDB:
			idMap.IMDB = link.Id
		case LinkTypeTMDB:
			idMap.TMDB = link.Id
		}
	}
	return &idMap
}
