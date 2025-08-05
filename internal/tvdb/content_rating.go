package tvdb

type ContentRating struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	Description string `json:"description"`
	ContentType string `json:"contentType"` // movie / episode
	Order       int    `json:"order"`
	Fullname    string `json:"fullname"`
}
