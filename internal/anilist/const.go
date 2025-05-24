package anilist

type Genre = string

const (
	GenreAction        Genre = "Action"
	GenreAdventure     Genre = "Adventure"
	GenreComedy        Genre = "Comedy"
	GenreDrama         Genre = "Drama"
	GenreEcchi         Genre = "Ecchi"
	GenreFantasy       Genre = "Fantasy"
	GenreHentai        Genre = "Hentai"
	GenreHorror        Genre = "Horror"
	GenreMahouShoujo   Genre = "Mahou Shoujo"
	GenreMecha         Genre = "Mecha"
	GenreMusic         Genre = "Music"
	GenreMystery       Genre = "Mystery"
	GenrePsychological Genre = "Psychological"
	GenreRomance       Genre = "Romance"
	GenreSciFi         Genre = "Sci-Fi"
	GenreSliceOfLife   Genre = "Slice of Life"
	GenreSports        Genre = "Sports"
	GenreSupernatural  Genre = "Supernatural"
)

var Genres = []Genre{
	GenreAction,
	GenreAdventure,
	GenreComedy,
	GenreDrama,
	GenreEcchi,
	GenreFantasy,
	GenreHentai,
	GenreHorror,
	GenreMahouShoujo,
	GenreMecha,
	GenreMusic,
	GenreMystery,
	GenrePsychological,
	GenreRomance,
	GenreSciFi,
	GenreSliceOfLife,
	GenreSports,
	GenreSupernatural,
}
