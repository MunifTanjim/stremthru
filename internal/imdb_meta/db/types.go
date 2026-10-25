package db

import "github.com/MunifTanjim/stremthru/plugin/imdb"

// Shared wire/storage shapes live in the public imdbplugin package so the
// external plugin binary can import them. Aliased here for convenience.
type (
	Person       = imdbplugin.Person
	Aka          = imdbplugin.Aka
	Title        = imdbplugin.Title
	TitleSummary = imdbplugin.TitleSummary
	Season       = imdbplugin.Season
	Episode      = imdbplugin.Episode
	Image        = imdbplugin.Image
	FullTitle    = imdbplugin.FullTitle
)
