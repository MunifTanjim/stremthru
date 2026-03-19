package serializd

type SeasonSummary struct {
	ID           int    `json:"id"`      // TMDBID
	AirDate      string `json:"airDate"` // YYYY-MM-DD
	EpisodeCount int    `json:"episodeCount,omitempty"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"posterPath"`
	SeasonID     int    `json:"seasonId,omitempty"`
	SeasonNumber int    `json:"seasonNumber"`
}

type ShowSummary struct {
	ID          int             `json:"id"` // TMDBID
	Name        string          `json:"name"`
	BannerImage string          `json:"bannerImage"`
	Seasons     []SeasonSummary `json:"seasons"`
	NumEpisodes int             `json:"numEpisodes"`
	IsPromoted  bool            `json:"isPromoted"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Network struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Show struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Tagline            string          `json:"tagline"`
	Summary            string          `json:"summary"`
	Status             string          `json:"status"`
	BannerImage        string          `json:"bannerImage"`
	PremiereDate       string          `json:"premiereDate"`
	LastAirDate        string          `json:"lastAirDate"`
	Networks           []Network       `json:"networks"`
	Genres             []Genre         `json:"genres"`
	Seasons            []SeasonSummary `json:"seasons"`
	NumSeasons         int             `json:"numSeasons"`
	NumEpisodes        int             `json:"numEpisodes"`
	NextEpisodeToAir   any             `json:"nextEpisodeToAir"`
	EpisodeToPreview   any             `json:"episodeToPreview"`
	EpisodeRunTime     []int           `json:"episodeRunTime"`
	NextEpisodeForUser any             `json:"nextEpisodeForUser"`
	IsPromoted         bool            `json:"isPromoted"`
}
