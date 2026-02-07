package newznab

import (
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Category = znab.Category

const (
	CustomCategoryOffset = 100000
)

var (
	CategoryReserved = Category{ID: 0, Name: "Reserved"}

	CategoryConsole = Category{ID: 1000, Name: "Console"}

	CategoryMovies = Category{ID: 2000, Name: "Movies"}

	CategoryAudio = Category{ID: 3000, Name: "Audio"}

	CategoryPC = Category{ID: 4000, Name: "PC"}

	CategoryTV = Category{ID: 5000, Name: "TV"}

	CategoryXXX = Category{ID: 6000, Name: "XXX"}

	CategoryBooks = Category{ID: 7000, Name: "Books"}

	CategoryOther = Category{ID: 8000, Name: "Other"}

	CategoryCustom = Category{ID: CustomCategoryOffset, Name: "Custom"}
)

func ParentCategory(c Category) Category {
	switch {
	case c.ID < 1000:
		return CategoryReserved
	case c.ID < 2000:
		return CategoryConsole
	case c.ID < 3000:
		return CategoryMovies
	case c.ID < 4000:
		return CategoryAudio
	case c.ID < 5000:
		return CategoryPC
	case c.ID < 6000:
		return CategoryTV
	case c.ID < 7000:
		return CategoryXXX
	case c.ID < 8000:
		return CategoryBooks
	case c.ID < 9000:
		return CategoryOther
	default:
		return CategoryCustom
	}
}

type Categories = znab.Categories
