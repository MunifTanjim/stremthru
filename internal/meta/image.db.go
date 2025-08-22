package meta

type ImageType string

type MetaImage struct {
	Id         int       `json:"id"`
	Provider   Provider  `json:"provider"`
	Type       ImageType `json:"type"`
	ExternalId string    `json:"external_id"`
	URL        string    `json:"url"`
	Language   Language  `json:"lang"`
	Height     int       `json:"height"`
	Width      int       `json:"width"`
}
