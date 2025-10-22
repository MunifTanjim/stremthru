package meta_type

import (
	"time"
)

type ItemType string

const (
	ItemTypeMovie ItemType = "movie"
	ItemTypeShow  ItemType = "show"
	ItemTypeMixed ItemType = ""
)

type List struct {
	Provider    Provider  `json:"provider"`
	Id          string    `json:"id"`
	Slug        string    `json:"slug"`
	UserId      string    `json:"user_id"`
	UserSlug    string    `json:"user_slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ItemType    ItemType  `json:"item_type"`
	IsPrivate   bool      `json:"is_private"`
	IsPersonal  bool      `json:"is_personal"`
	ItemCount   int       `json:"item_count"`
	Version     int64     `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`

	Items []ListItem `json:"items"`
}

type ListItem struct {
	Type        ItemType  `json:"type"`
	Id          string    `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Year        int       `json:"year"`
	IsAdult     bool      `json:"is_adult"`
	Runtime     int       `json:"runtime"`
	Rating      int       `json:"rating"`
	Poster      string    `json:"poster"`
	UpdatedAt   time.Time `json:"updated_at"`
	Index       int       `json:"index"`
	GenreIds    []string  `json:"genre_ids"`
	IdMap       IdMap     `json:"id_map"`
}
