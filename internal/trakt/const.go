package trakt

type Genre = string

const (
	GenreAction          Genre = "action"
	GenreAdventure       Genre = "adventure"
	GenreAnimation       Genre = "animation"
	GenreAnime           Genre = "anime"
	GenreBiography       Genre = "biography"
	GenreChildren        Genre = "children"
	GenreComedy          Genre = "comedy"
	GenreCrime           Genre = "crime"
	GenreDocumentary     Genre = "documentary"
	GenreDonghua         Genre = "donghua"
	GenreDrama           Genre = "drama"
	GenreFamily          Genre = "family"
	GenreFantasy         Genre = "fantasy"
	GenreGameShow        Genre = "game-show"
	GenreHistory         Genre = "history"
	GenreHoliday         Genre = "holiday"
	GenreHomeAndGarden   Genre = "home-and-garden"
	GenreHorror          Genre = "horror"
	GenreMiniSeries      Genre = "mini-series"
	GenreMusic           Genre = "music"
	GenreMusical         Genre = "musical"
	GenreMystery         Genre = "mystery"
	GenreNews            Genre = "news"
	GenreNone            Genre = "none"
	GenreReality         Genre = "reality"
	GenreRomance         Genre = "romance"
	GenreScienceFiction  Genre = "science-fiction"
	GenreShort           Genre = "short"
	GenreSoap            Genre = "soap"
	GenreSpecialInterest Genre = "special-interest"
	GenreSportingEvent   Genre = "sporting-event"
	GenreSuperhero       Genre = "superhero"
	GenreSuspense        Genre = "suspense"
	GenreTalkShow        Genre = "talk-show"
	GenreThriller        Genre = "thriller"
	GenreWar             Genre = "war"
	GenreWestern         Genre = "western"
)

var genreNameById = map[Genre]string{
	GenreAction:          "Action",
	GenreAdventure:       "Adventure",
	GenreAnimation:       "Animation",
	GenreAnime:           "Anime",
	GenreBiography:       "Biography",
	GenreChildren:        "Children",
	GenreComedy:          "Comedy",
	GenreCrime:           "Crime",
	GenreDocumentary:     "Documentary",
	GenreDonghua:         "Donghua",
	GenreDrama:           "Drama",
	GenreFamily:          "Family",
	GenreFantasy:         "Fantasy",
	GenreGameShow:        "Game Show",
	GenreHistory:         "History",
	GenreHoliday:         "Holiday",
	GenreHomeAndGarden:   "Home & Garden",
	GenreHorror:          "Horror",
	GenreMiniSeries:      "Mini Series",
	GenreMusic:           "Music",
	GenreMusical:         "Musical",
	GenreMystery:         "Mystery",
	GenreNews:            "News",
	GenreNone:            "None",
	GenreReality:         "Reality",
	GenreRomance:         "Romance",
	GenreScienceFiction:  "Science Fiction",
	GenreShort:           "Short",
	GenreSoap:            "Soap",
	GenreSpecialInterest: "Special Interest",
	GenreSportingEvent:   "Sporting Event",
	GenreSuperhero:       "Superhero",
	GenreSuspense:        "Suspense",
	GenreTalkShow:        "Talk Show",
	GenreThriller:        "Thriller",
	GenreWar:             "War",
	GenreWestern:         "Western",
}

var genreIds = []Genre{
	GenreAction,
	GenreAdventure,
	GenreAnimation,
	GenreAnime,
	GenreBiography,
	GenreChildren,
	GenreComedy,
	GenreCrime,
	GenreDocumentary,
	GenreDonghua,
	GenreDrama,
	GenreFamily,
	GenreFantasy,
	GenreGameShow,
	GenreHistory,
	GenreHoliday,
	GenreHomeAndGarden,
	GenreHorror,
	GenreMiniSeries,
	GenreMusic,
	GenreMusical,
	GenreMystery,
	GenreNews,
	GenreNone,
	GenreReality,
	GenreRomance,
	GenreScienceFiction,
	GenreShort,
	GenreSoap,
	GenreSpecialInterest,
	GenreSportingEvent,
	GenreSuperhero,
	GenreSuspense,
	GenreTalkShow,
	GenreThriller,
	GenreWar,
	GenreWestern,
}

var movieGenreIds = []Genre{
	GenreAction,
	GenreAdventure,
	GenreAnimation,
	GenreAnime,
	GenreComedy,
	GenreCrime,
	GenreDocumentary,
	GenreDonghua,
	GenreDrama,
	GenreFamily,
	GenreFantasy,
	GenreHistory,
	GenreHoliday,
	GenreHorror,
	GenreMusic,
	GenreMusical,
	GenreMystery,
	GenreNone,
	GenreRomance,
	GenreScienceFiction,
	GenreShort,
	GenreSportingEvent,
	GenreSuperhero,
	GenreSuspense,
	GenreThriller,
	GenreWar,
	GenreWestern,
}

var showGenreIds = []Genre{
	GenreAction,
	GenreAdventure,
	GenreAnimation,
	GenreAnime,
	GenreBiography,
	GenreChildren,
	GenreComedy,
	GenreCrime,
	GenreDocumentary,
	GenreDonghua,
	GenreDrama,
	GenreFamily,
	GenreFantasy,
	GenreGameShow,
	GenreHistory,
	GenreHoliday,
	GenreHomeAndGarden,
	GenreHorror,
	GenreMiniSeries,
	GenreMusic,
	GenreMusical,
	GenreMystery,
	GenreNews,
	GenreNone,
	GenreReality,
	GenreRomance,
	GenreScienceFiction,
	GenreShort,
	GenreSoap,
	GenreSpecialInterest,
	GenreSportingEvent,
	GenreSuperhero,
	GenreSuspense,
	GenreTalkShow,
	GenreThriller,
	GenreWar,
	GenreWestern,
}

var GenreNames, MovieGenreNames, ShowGenreNames, genreIdByName = func() ([]string, []string, []string, map[string]Genre) {
	idByName := make(map[string]Genre, len(genreNameById))

	names := make([]string, len(genreIds))
	for i, genre := range genreIds {
		names[i] = genreNameById[genre]
		idByName[names[i]] = genre
	}

	movieNames := make([]string, len(movieGenreIds))
	for i, genre := range movieGenreIds {
		movieNames[i] = genreNameById[genre]
	}

	showNames := make([]string, len(showGenreIds))
	for i, genre := range showGenreIds {
		showNames[i] = genreNameById[genre]
	}

	return names, movieNames, showNames, idByName
}()
