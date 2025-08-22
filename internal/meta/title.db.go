package meta

import "github.com/MunifTanjim/stremthru/internal/db"

type MetaTitle struct {
	Id        int          `json:"id"`
	Provider  Provider     `json:"provider"`
	Language  Language     `json:"lang"`
	Value     string       `json:"value"`
	UpdatedAt db.Timestamp `json:"uat"`
}
