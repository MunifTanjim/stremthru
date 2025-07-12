package tmdb

import "slices"

const IMAGE_BASE_URL = "https://image.tmdb.org/t/p/"

type BackdropSize string

const (
	BackdropSizeW300     BackdropSize = "w300"
	BackdropSizeW780     BackdropSize = "w780"
	BackdropSizeW1280    BackdropSize = "w1280"
	BackdropSizeOriginal BackdropSize = "original"
)

type PosterSize string

const (
	PosterSizeW92      PosterSize = "w92"
	PosterSizeW154     PosterSize = "w154"
	PosterSizeW185     PosterSize = "w185"
	PosterSizeW342     PosterSize = "w342"
	PosterSizeW500     PosterSize = "w500"
	PosterSizeW780     PosterSize = "w780"
	PosterSizeOriginal PosterSize = "original"
)

var movieGenreMap = map[int]string{
	12:    "Adventure",
	14:    "Fantasy",
	16:    "Animation",
	18:    "Drama",
	27:    "Horror",
	28:    "Action",
	35:    "Comedy",
	36:    "History",
	37:    "Western",
	53:    "Thriller",
	80:    "Crime",
	99:    "Documentary",
	878:   "Science Fiction",
	9648:  "Mystery",
	10402: "Music",
	10749: "Romance",
	10751: "Family",
	10752: "War",
	10770: "TV Movie",
}

var tvGenreMap = map[int]string{
	16:    "Animation",
	18:    "Drama",
	35:    "Comedy",
	37:    "Western",
	80:    "Crime",
	99:    "Documentary",
	9648:  "Mystery",
	10751: "Family",
	10759: "Action & Adventure",
	10762: "Kids",
	10763: "News",
	10764: "Reality",
	10765: "Sci-Fi & Fantasy",
	10766: "Soap",
	10767: "Talk",
	10768: "War & Politics",
}

type Genre string

var Genres, MovieGenres, TVGenres, genreIdByName, genreNameById = func() ([]string, []string, []string, map[string]int, map[int]string) {
	genres := make([]string, 0, len(movieGenreMap)+len(tvGenreMap))
	idByName := map[string]int{}
	nameById := map[int]string{}

	movieGenres := make([]string, 0, len(movieGenreMap))
	for id, genre := range movieGenreMap {
		genres = append(genres, genre)
		idByName[genre] = id
		nameById[id] = genre
		movieGenres = append(movieGenres, genre)
	}
	slices.Sort(movieGenres)

	tvGenres := make([]string, 0, len(tvGenreMap))
	for id, genre := range tvGenreMap {
		if _, seen := nameById[id]; !seen {
			genres = append(genres, genre)
		}
		idByName[genre] = id
		nameById[id] = genre
		tvGenres = append(tvGenres, genre)
	}
	slices.Sort(tvGenres)

	slices.Sort(genres)

	return genres, movieGenres, tvGenres, idByName, nameById
}()

func (genre Genre) Id() int {
	id, ok := genreIdByName[string(genre)]
	if ok {
		return id
	}
	return 0
}
