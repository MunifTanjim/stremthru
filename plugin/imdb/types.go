// Package imdbplugin is the shared contract between stremthru and the
// external IMDb provider plugin binary ($STREMTHRU_PLUGIN_DIR/imdb).
//
// Plugin authors: implement Provider and call Serve in main:
//
//	package main
//
//	import (
//		"context"
//
//		imdbplugin "github.com/MunifTanjim/stremthru/plugin/imdb"
//	)
//
//	type provider struct{}
//
//	func (p *provider) GetTitle(ctx context.Context, tid string) (*imdbplugin.FullTitle, error) {
//		// fetch from IMDb APIs, shape into imdbplugin.FullTitle
//	}
//
//	func (p *provider) GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error) {
//		// fetch one page of the title's images, starting after the cursor
//	}
//
//	func (p *provider) GetEpisodes(ctx context.Context, ids []string) ([]imdbplugin.TitleSummary, error) {
//		// fetch summaries for at most imdbplugin.MaxEpisodeBatch episode ids
//	}
//
//	func (p *provider) GetEpisodeList(ctx context.Context, id string) ([]imdbplugin.Season, error) {
//		// fetch the series' seasons with bare episodes
//	}
//
//	func (p *provider) GetRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error) {
//		// fetch ratings for at most imdbplugin.MaxRatingBatch title ids
//	}
//
//	func main() {
//		imdbplugin.Serve(&provider{})
//	}
package imdbplugin

import "time"

type Person struct {
	NMID      string `json:"nmid"`
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Pos       int    `json:"-"`
}

type Aka struct {
	Text     string `json:"text"`
	Country  string `json:"country,omitempty"`
	Language string `json:"language,omitempty"`
}

// TitleSummary is what a title and an episode have in common. Episodes are
// titles too: the host stores one as a tvEpisode title.
type TitleSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
	Runtime     int    `json:"runtime,omitempty"` // seconds
	Image       Image  `json:"image,omitempty"`
	Rating      int    `json:"rating"` // one-decimal rating x10 (77 = 7.7), 0 = unrated
	Votes       int    `json:"votes"`
}

type Title struct {
	TitleSummary
	Type           string   `json:"type"`
	IsSeries       bool     `json:"is_series"`
	OrigName       string   `json:"orig_name,omitempty"`
	YearStart      int      `json:"year_start"`
	YearEnd        *int     `json:"year_end,omitempty"`
	ReleaseCountry string   `json:"release_country,omitempty"`
	TrailerVIs     []string `json:"trailer_vi_ids"`
	Certificate    string   `json:"certificate,omitempty"`
	// Partial is set by the host, not the plugin: the title is an episode
	// stored from its series' payload and holds only the fields that carries.
	Partial   bool      `json:"partial,omitempty"`
	Genres    []string  `json:"genres"`
	Directors []Person  `json:"directors"`
	Cast      []Person  `json:"cast"`
	Akas      []Aka     `json:"akas"`
	UpdatedAt time.Time `json:"uat"`
}

type Season struct {
	Number int `json:"number"`
	// Episodes are ordered by episode number (0 is a real number, used for
	// specials that air before episode 1; unnumbered episodes go last), then
	// by release date (undated last).
	Episodes []Episode `json:"episodes"`
}

type Episode struct {
	TitleSummary
	// Episode is nil when IMDb gives the episode no number.
	Episode *int `json:"episode,omitempty"`
}

// Rating is a title's rating and vote count. Rating uses the same encoding
// as TitleSummary.Rating.
type Rating struct {
	ID     string `json:"id"`
	Rating int    `json:"rating"`
	Votes  int    `json:"votes"`
}

// ImagePage is one page of a title's images.
type ImagePage struct {
	Images []Image `json:"images"`
	// Next is the cursor for the following page, empty on the last page.
	Next string `json:"next,omitempty"`
}

type FullTitle struct {
	Title   Title    `json:"title"`
	Seasons []Season `json:"seasons"`
}

// Image is an IMDb image: the rm image id plus display data. The URL cannot
// be rebuilt from the id alone (media paths use a different hash), so all
// parts are stored. It is used for a title's or episode's primary image and
// for gallery entries; in the gallery, title-level images have an empty EpID
// and episode stills carry the episode id. Fields the source does not provide
// (a primary image has no kind or caption) are omitted.
type Image struct {
	ID      string `json:"id"`
	EpID    string `json:"ep_id,omitempty"`
	Kind    string `json:"kind,omitempty"`
	URL     string `json:"url,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Caption string `json:"caption,omitempty"`
	Pos     int    `json:"-"`
}
