package letterboxd

const SITE_BASE_URL = "https://letterboxd.com"

var genreNameById = map[string]string{
	"8G":  "Action",
	"9k":  "Adventure",
	"8m":  "Animation",
	"7I":  "Comedy",
	"9Y":  "Crime",
	"ai":  "Documentary",
	"7S":  "Drama",
	"8w":  "Family",
	"82":  "Fantasy",
	"90":  "History",
	"aC":  "Horror",
	"b6":  "Music",
	"aW":  "Mystery",
	"8c":  "Romance",
	"9a":  "Science Fiction",
	"a8":  "Thriller",
	"1hO": "TV Movie",
	"9u":  "War",
	"8Q":  "Western",
}

var GenreNames = func() []string {
	genreNames := make([]string, 0, len(genreNameById))
	for _, name := range genreNameById {
		genreNames = append(genreNames, name)
	}
	return genreNames
}()
