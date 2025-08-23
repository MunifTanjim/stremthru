package mdblist

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
	GenreDrama           Genre = "drama"
	GenreFamily          Genre = "family"
	GenreFantasy         Genre = "fantasy"
	GenreFilmNoir        Genre = "film-noir"
	GenreGameShow        Genre = "game-show"
	GenreHistory         Genre = "history"
	GenreHoliday         Genre = "holiday"
	GenreHomeAndGarden   Genre = "home-and-garden"
	GenreHorror          Genre = "horror"
	GenreMusic           Genre = "music"
	GenreMusical         Genre = "musical"
	GenreMystery         Genre = "mystery"
	GenreNews            Genre = "news"
	GenreReality         Genre = "reality"
	GenreRealityTV       Genre = "reality-tv"
	GenreRomance         Genre = "romance"
	GenreScienceFiction  Genre = "science-fiction"
	GenreSciFi           Genre = "sci-fi"
	GenreShort           Genre = "short"
	GenreSoap            Genre = "soap"
	GenreSpecialInterest Genre = "special-interest"
	GenreSport           Genre = "sport"
	GenreSportingEvent   Genre = "sporting-event"
	GenreSuperhero       Genre = "superhero"
	GenreSuspense        Genre = "suspense"
	GenreTalkShow        Genre = "talk-show"
	GenreThriller        Genre = "thriller"
	GenreTVMovie         Genre = "tv-movie"
	GenreWar             Genre = "war"
	GenreWestern         Genre = "western"

	GenreAnimeBoysLove      Genre = "anime-bl"
	GenreAnimeHistorical    Genre = "anime-historical"
	GenreAnimeIsekai        Genre = "anime-isekai"
	GenreAnimeJosei         Genre = "anime-josei"
	GenreAnimeMartialArts   Genre = "anime-martial-arts"
	GenreAnimeMecha         Genre = "anime-mecha"
	GenreAnimeMilitary      Genre = "anime-military"
	GenreAnimeMusic         Genre = "anime-music"
	GenreAnimeParody        Genre = "anime-parody"
	GenreAnimePsychological Genre = "anime-psychological"
	GenreAnimeSamurai       Genre = "anime-samurai"
	GenreAnimeSchool        Genre = "anime-school"
	GenreAnimeSeinen        Genre = "anime-seinen"
	GenreAnimeShoujo        Genre = "anime-shoujo"
	GenreAnimeShounen       Genre = "anime-shounen"
	GenreAnimeSliceOfLife   Genre = "anime-slice-of-life"
	GenreAnimeSpace         Genre = "anime-space"
	GenreAnimeSports        Genre = "anime-sports"
	GenreAnimeSupernatural  Genre = "anime-supernatural"
	GenreAnimeVampire       Genre = "anime-vampire"
	GenreAnimeYuri          Genre = "anime-yuri"
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
	GenreDrama:           "Drama",
	GenreFamily:          "Family",
	GenreFantasy:         "Fantasy",
	GenreFilmNoir:        "Film Noir",
	GenreGameShow:        "Game Show",
	GenreHistory:         "History",
	GenreHoliday:         "Holiday",
	GenreHomeAndGarden:   "Home and Garden",
	GenreHorror:          "Horror",
	GenreMusic:           "Music",
	GenreMusical:         "Musical",
	GenreMystery:         "Mystery",
	GenreNews:            "News",
	GenreReality:         "Reality",
	GenreRealityTV:       "Reality TV",
	GenreRomance:         "Romance",
	GenreScienceFiction:  "Science Fiction",
	GenreSciFi:           "Sci-Fi",
	GenreShort:           "Short",
	GenreSoap:            "Soap",
	GenreSpecialInterest: "Special Interest",
	GenreSport:           "Sport",
	GenreSportingEvent:   "Sporting Event",
	GenreSuperhero:       "Superhero",
	GenreSuspense:        "Suspense",
	GenreTalkShow:        "Talk Show",
	GenreThriller:        "Thriller",
	GenreTVMovie:         "TV Movie",
	GenreWar:             "War",
	GenreWestern:         "Western",

	GenreAnimeBoysLove:      "Anime: Boys' Love (Yaio)",
	GenreAnimeHistorical:    "Anime: Historical",
	GenreAnimeIsekai:        "Anime: Isekai",
	GenreAnimeJosei:         "Anime: Josei",
	GenreAnimeMartialArts:   "Anime: Martial Arts",
	GenreAnimeMecha:         "Anime: Mecha",
	GenreAnimeMilitary:      "Anime: Military",
	GenreAnimeMusic:         "Anime: Music",
	GenreAnimeParody:        "Anime: Parody",
	GenreAnimePsychological: "Anime: Psychological",
	GenreAnimeSamurai:       "Anime: Samurai",
	GenreAnimeSchool:        "Anime: School",
	GenreAnimeSeinen:        "Anime: Seinen",
	GenreAnimeShoujo:        "Anime: Shoujo",
	GenreAnimeShounen:       "Anime: Shounen",
	GenreAnimeSliceOfLife:   "Anime: Slice of Life",
	GenreAnimeSpace:         "Anime: Space",
	GenreAnimeSports:        "Anime: Sports",
	GenreAnimeSupernatural:  "Anime: Supernatural",
	GenreAnimeVampire:       "Anime: Vampire",
	GenreAnimeYuri:          "Anime: Yuri",
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
	GenreDrama,
	GenreFamily,
	GenreFantasy,
	GenreFilmNoir,
	GenreGameShow,
	GenreHistory,
	GenreHoliday,
	GenreHomeAndGarden,
	GenreHorror,
	GenreMusic,
	GenreMusical,
	GenreMystery,
	GenreNews,
	GenreReality,
	GenreRealityTV,
	GenreRomance,
	GenreScienceFiction,
	GenreSciFi,
	GenreShort,
	GenreSoap,
	GenreSpecialInterest,
	GenreSport,
	GenreSportingEvent,
	GenreSuperhero,
	GenreSuspense,
	GenreTalkShow,
	GenreThriller,
	GenreTVMovie,
	GenreWar,
	GenreWestern,
}

var animeGenreIds = []Genre{
	GenreAnimeBoysLove,
	GenreAnimeHistorical,
	GenreAnimeIsekai,
	GenreAnimeJosei,
	GenreAnimeMartialArts,
	GenreAnimeMecha,
	GenreAnimeMilitary,
	GenreAnimeMusic,
	GenreAnimeParody,
	GenreAnimePsychological,
	GenreAnimeSamurai,
	GenreAnimeSchool,
	GenreAnimeSeinen,
	GenreAnimeShoujo,
	GenreAnimeShounen,
	GenreAnimeSliceOfLife,
	GenreAnimeSpace,
	GenreAnimeSports,
	GenreAnimeSupernatural,
	GenreAnimeVampire,
	GenreAnimeYuri,
}

var GenreNames, AnimeGenreNames = func() ([]string, []string) {
	names := make([]string, len(genreIds))
	for i, genre := range genreIds {
		names[i] = genreNameById[genre]
	}

	animeNames := make([]string, len(animeGenreIds))
	for i, genre := range animeGenreIds {
		animeNames[i] = genreNameById[genre]
	}

	return names, animeNames
}()
