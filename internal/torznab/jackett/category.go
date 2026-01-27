package jackett

import (
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Category = znab.Category

const (
	CustomCategoryOffset = 100000
)

// https://github.com/Jackett/Jackett/wiki/Jackett-Categories
var (
	CategoryReserved = Category{ID: 0000, Name: "Reserved"}

	CategoryConsole            = Category{ID: 1000, Name: "Console"}
	CategoryConsole_NDS        = Category{ID: 1010, Name: "Console/NDS"}
	CategoryConsole_PSP        = Category{ID: 1020, Name: "Console/PSP"}
	CategoryConsole_Wii        = Category{ID: 1030, Name: "Console/Wii"}
	CategoryConsole_XBox       = Category{ID: 1040, Name: "Console/XBox"}
	CategoryConsole_XBox360    = Category{ID: 1050, Name: "Console/XBox 360"}
	CategoryConsole_Wiiware    = Category{ID: 1060, Name: "Console/Wiiware"}
	CategoryConsole_XBox360DLC = Category{ID: 1070, Name: "Console/XBox 360 DLC"}
	CategoryConsole_PS3        = Category{ID: 1080, Name: "Console/PS3"}
	CategoryConsole_Other      = Category{ID: 1090, Name: "Console/Other"}
	CategoryConsole_3DS        = Category{ID: 1110, Name: "Console/3DS"}
	CategoryConsole_PSVita     = Category{ID: 1120, Name: "Console/PS Vita"}
	CategoryConsole_WiiU       = Category{ID: 1130, Name: "Console/WiiU"}
	CategoryConsole_XBOXOne    = Category{ID: 1140, Name: "Console/XBox One"}
	CategoryConsole_PS4        = Category{ID: 1180, Name: "Console/PS4"}

	CategoryMovies         = Category{ID: 2000, Name: "Movies"}
	CategoryMovies_Foreign = Category{ID: 2010, Name: "Movies/Foreign"}
	CategoryMovies_Other   = Category{ID: 2020, Name: "Movies/Other"}
	CategoryMovies_SD      = Category{ID: 2030, Name: "Movies/SD"}
	CategoryMovies_HD      = Category{ID: 2040, Name: "Movies/HD"}
	CategoryMovies_UHD     = Category{ID: 2045, Name: "Movies/UHD"}
	CategoryMovies_BluRay  = Category{ID: 2050, Name: "Movies/BluRay"}
	CategoryMovies_3D      = Category{ID: 2060, Name: "Movies/3D"}
	CategoryMovies_DVD     = Category{ID: 2070, Name: "Movies/DVD"}
	CategoryMovies_WEBDL   = Category{ID: 2080, Name: "Movies/WEB-DL"}

	CategoryAudio           = Category{ID: 3000, Name: "Audio"}
	CategoryAudio_MP3       = Category{ID: 3010, Name: "Audio/MP3"}
	CategoryAudio_Video     = Category{ID: 3020, Name: "Audio/Video"}
	CategoryAudio_Audiobook = Category{ID: 3030, Name: "Audio/Audiobook"}
	CategoryAudio_Lossless  = Category{ID: 3040, Name: "Audio/Lossless"}
	CategoryAudio_Other     = Category{ID: 3050, Name: "Audio/Other"}
	CategoryAudio_Foreign   = Category{ID: 3060, Name: "Audio/Foreign"}

	CategoryPC               = Category{ID: 4000, Name: "PC"}
	CategoryPC_0day          = Category{ID: 4010, Name: "PC/0day"}
	CategoryPC_ISO           = Category{ID: 4020, Name: "PC/ISO"}
	CategoryPC_Mac           = Category{ID: 4030, Name: "PC/Mac"}
	CategoryPC_MobileOther   = Category{ID: 4040, Name: "PC/Mobile-Other"}
	CategoryPC_Games         = Category{ID: 4050, Name: "PC/Games"}
	CategoryPC_MobileIOS     = Category{ID: 4060, Name: "PC/Mobile-iOS"}
	CategoryPC_MobileAndroid = Category{ID: 4070, Name: "PC/Mobile-Android"}

	CategoryTV             = Category{ID: 5000, Name: "TV"}
	CategoryTV_WEBDL       = Category{ID: 5010, Name: "TV/WEB-DL"}
	CategoryTV_Foreign     = Category{ID: 5020, Name: "TV/Foreign"}
	CategoryTV_SD          = Category{ID: 5030, Name: "TV/SD"}
	CategoryTV_HD          = Category{ID: 5040, Name: "TV/HD"}
	CategoryTV_UHD         = Category{ID: 5045, Name: "TV/UHD"}
	CategoryTV_Other       = Category{ID: 5050, Name: "TV/Other"}
	CategoryTV_Sport       = Category{ID: 5060, Name: "TV/Sport"}
	CategoryTV_Anime       = Category{ID: 5070, Name: "TV/Anime"}
	CategoryTV_Documentary = Category{ID: 5080, Name: "TV/Documentary"}

	CategoryXXX          = Category{ID: 6000, Name: "XXX"}
	CategoryXXX_DVD      = Category{ID: 6010, Name: "XXX/DVD"}
	CategoryXXX_WMV      = Category{ID: 6020, Name: "XXX/WMV"}
	CategoryXXX_XviD     = Category{ID: 6030, Name: "XXX/XviD"}
	CategoryXXX_x264     = Category{ID: 6040, Name: "XXX/x264"}
	CategoryXXX_UHD      = Category{ID: 6045, Name: "XXX/UHD"}
	CategoryXXX_Pack     = Category{ID: 6050, Name: "XXX/Pack"}
	CategoryXXX_ImageSet = Category{ID: 6060, Name: "XXX/ImageSet"}
	CategoryXXX_Other    = Category{ID: 6070, Name: "XXX/Other"}
	CategoryXXX_SD       = Category{ID: 6080, Name: "XXX/SD"}
	CategoryXXX_WEBDL    = Category{ID: 6090, Name: "XXX/WEB-DL"}

	CategoryBooks           = Category{ID: 7000, Name: "Books"}
	CategoryBooks_Mags      = Category{ID: 7010, Name: "Books/Mags"}
	CategoryBooks_EBook     = Category{ID: 7020, Name: "Books/EBook"}
	CategoryBooks_Comics    = Category{ID: 7030, Name: "Books/Comics"}
	CategoryBooks_Technical = Category{ID: 7040, Name: "Books/Technical"}
	CategoryBooks_Other     = Category{ID: 7050, Name: "Books/Other"}
	CategoryBooks_Foreign   = Category{ID: 7060, Name: "Books/Foreign"}

	CategoryOther        = Category{ID: 8000, Name: "Other"}
	CategoryOther_Misc   = Category{ID: 8010, Name: "Other/Misc"}
	CategoryOther_Hashed = Category{ID: 8020, Name: "Other/Hashed"}
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
	}
	return CategoryOther
}

var AllCategories = znab.Categories{
	CategoryConsole,
	CategoryConsole_NDS,
	CategoryConsole_PSP,
	CategoryConsole_Wii,
	CategoryConsole_XBox,
	CategoryConsole_XBox360,
	CategoryConsole_Wiiware,
	CategoryConsole_XBox360DLC,
	CategoryConsole_PS3,
	CategoryConsole_Other,
	CategoryConsole_3DS,
	CategoryConsole_PSVita,
	CategoryConsole_WiiU,
	CategoryConsole_XBOXOne,
	CategoryConsole_PS4,
	CategoryMovies,
	CategoryMovies_Foreign,
	CategoryMovies_Other,
	CategoryMovies_SD,
	CategoryMovies_HD,
	CategoryMovies_UHD,
	CategoryMovies_BluRay,
	CategoryMovies_3D,
	CategoryMovies_DVD,
	CategoryMovies_WEBDL,
	CategoryAudio,
	CategoryAudio_MP3,
	CategoryAudio_Video,
	CategoryAudio_Audiobook,
	CategoryAudio_Lossless,
	CategoryAudio_Other,
	CategoryAudio_Foreign,
	CategoryPC,
	CategoryPC_0day,
	CategoryPC_ISO,
	CategoryPC_Mac,
	CategoryPC_MobileOther,
	CategoryPC_Games,
	CategoryPC_MobileIOS,
	CategoryPC_MobileAndroid,
	CategoryTV,
	CategoryTV_WEBDL,
	CategoryTV_Foreign,
	CategoryTV_SD,
	CategoryTV_HD,
	CategoryTV_UHD,
	CategoryTV_Other,
	CategoryTV_Sport,
	CategoryTV_Anime,
	CategoryTV_Documentary,
	CategoryXXX,
	CategoryXXX_DVD,
	CategoryXXX_WMV,
	CategoryXXX_XviD,
	CategoryXXX_x264,
	CategoryXXX_UHD,
	CategoryXXX_Pack,
	CategoryXXX_ImageSet,
	CategoryXXX_Other,
	CategoryXXX_SD,
	CategoryXXX_WEBDL,
	CategoryBooks,
	CategoryBooks_Mags,
	CategoryBooks_EBook,
	CategoryBooks_Comics,
	CategoryBooks_Technical,
	CategoryBooks_Other,
	CategoryBooks_Foreign,
	CategoryOther,
	CategoryOther_Misc,
	CategoryOther_Hashed,
}

var categoryById = func() map[int]*Category {
	m := make(map[int]*Category, len(AllCategories))
	for i := range AllCategories {
		cat := &AllCategories[i]
		m[cat.ID] = cat
	}
	return m
}()

func GetCategoryById(id int) *Category {
	if cat, ok := categoryById[id]; ok {
		return cat
	}
	return nil
}
