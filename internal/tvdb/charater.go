package tvdb

type Character struct {
	Id                   int        `json:"id"`
	Name                 string     `json:"name"`
	PeopleId             int        `json:"peopleId"`
	SeriesId             int        `json:"seriesId"`
	Series               any        `json:"series"`
	Movie                any        `json:"movie"`
	MovieId              int        `json:"movieId"`
	EpisodeId            int        `json:"episodeId"`
	Type                 int        `json:"type"`
	Image                string     `json:"image"`
	Sort                 int        `json:"sort"`
	IsFeatured           bool       `json:"isFeatured"`
	URL                  string     `json:"url"` // slug
	NameTranslations     []string   `json:"nameTranslations"`
	OverviewTranslations []string   `json:"overviewTranslations"`
	Aliases              []Alias    `json:"aliases"`
	PeopleType           PeopleType `json:"peopleType"`
	PersonName           string     `json:"personName"`
	TagOptions           any        `json:"tagOptions"`
	PersonImgURL         string     `json:"personImgUrl"`
}

type Characters []Character

func (chars Characters) GetDirectors() []Character {
	directors := []Character{}
	for i := range chars {
		char := &chars[i]
		if char.PeopleType == PeopleTypeDirector {
			directors = append(directors, *char)
		}
	}
	return directors
}

func (chars Characters) GetActors() []Character {
	directors := []Character{}
	for i := range chars {
		char := &chars[i]
		if char.PeopleType == PeopleTypeActor {
			directors = append(directors, *char)
		}
	}
	return directors
}
