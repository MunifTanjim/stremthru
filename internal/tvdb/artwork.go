package tvdb

const ArtworkBaseURL = "https://artworks.thetvdb.com"

type ArtworkType int

const (
	ArtworkTypeSeriesBanner        ArtworkType = 1
	ArtworkTypeSeriesPoster        ArtworkType = 2
	ArtworkTypeSeriesBackground    ArtworkType = 3
	ArtworkTypeSeriesIcon          ArtworkType = 5
	ArtworkTypeSeasonBanner        ArtworkType = 6
	ArtworkTypeSeasonPoster        ArtworkType = 7
	ArtworkTypeSeasonBackground    ArtworkType = 8
	ArtworkTypeSeasonIcon          ArtworkType = 10
	ArtworkTypeEpisodeScreencap169 ArtworkType = 11
	ArtworkTypeEpisodeScreencap43  ArtworkType = 12
	ArtworkTypeActorPhoto          ArtworkType = 13
	ArtworkTypeMoviePoster         ArtworkType = 14
	ArtworkTypeMovieBackground     ArtworkType = 15
	ArtworkTypeMovieBanner         ArtworkType = 16
	ArtworkTypeMovieIcon           ArtworkType = 18
	ArtworkTypeCompanyIcon         ArtworkType = 19
	ArtworkTypeSeriesCinemagraph   ArtworkType = 20
	ArtworkTypeMovieCinemagraph    ArtworkType = 21
	ArtworkTypeSeriesClearArt      ArtworkType = 22
	ArtworkTypeSeriesClearLogo     ArtworkType = 23
	ArtworkTypeMovieClearArt       ArtworkType = 24
	ArtworkTypeMovieClearLogo      ArtworkType = 25
	ArtworkTypeAwardIcon           ArtworkType = 26
	ArtworkTypeListPoster          ArtworkType = 27
)

type Artwork struct {
	Id           int         `json:"id"`
	Image        string      `json:"image"`
	Thumbnail    string      `json:"thumbnail"`
	Language     string      `json:"language"`
	Type         ArtworkType `json:"type"`
	Score        int64       `json:"score"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	IncludesText bool        `json:"includesText"`
}

type ArtworkStatus int

const (
	ArtworkStatusLowQuality           ArtworkStatus = 1
	ArtworkStatusImproperActionShot   ArtworkStatus = 2
	ArtworkStatusSpoiler              ArtworkStatus = 3
	ArtworkStatusAdultContent         ArtworkStatus = 4
	ArtworkStatusAutomaticallyResized ArtworkStatus = 5
)

type ExtendedArtwork struct {
	Artwork
	ThumbnailWidth  int   `json:"thumbnailWidth"`
	ThumbnailHeight int   `json:"thumbnailHeight"`
	UpdatedAt       int64 `json:"updatedAt"`
	Status          struct {
		Id   ArtworkStatus `json:"id"`
		Name string        `json:"name"`
	} `json:"status"`
	TagOptions []TagOption `json:"tagOptions"`
	SeriesId   int         `json:"seriesId,omitempty"`
}
