package tvdb

type Genre struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

var genreNameById = map[int]string{
	1:  "Soap",
	2:  "Science Fiction",
	3:  "Reality",
	4:  "News",
	5:  "Mini-Series",
	6:  "Horror",
	7:  "Home and Garden",
	8:  "Game Show",
	9:  "Food",
	10: "Fantasy",
	11: "Family",
	12: "Drama",
	13: "Documentary",
	14: "Crime",
	15: "Comedy",
	16: "Children",
	17: "Animation",
	18: "Adventure",
	19: "Action",
	21: "Sport",
	22: "Suspense",
	23: "Talk Show",
	24: "Thriller",
	25: "Travel",
	26: "Western",
	27: "Anime",
	28: "Romance",
	29: "Musical",
	30: "Podcast",
	31: "Mystery",
	32: "Indie",
	33: "History",
	34: "War",
	35: "Martial Arts",
	36: "Awards Show",
}

var GenreNames, genreIdByName = func() ([]string, map[string]int) {
	genreNames := make([]string, 0, len(genreNameById))
	idByName := make(map[string]int, len(genreNameById))

	for id, name := range genreNameById {
		genreNames = append(genreNames, name)
		idByName[name] = id
	}

	return genreNames, idByName
}()

func GenreId(genre string) int {
	id, ok := genreIdByName[genre]
	if ok {
		return id
	}
	return 0
}
