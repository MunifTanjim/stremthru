package torznab

import (
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Category = znab.Category

const (
	CustomCategoryOffset = 100000
)

// Categories from the Newznab spec
// https://github.com/nZEDb/nZEDb/blob/main/docs/newznab_api_specification.txt#L627
var (
	CategoryOther              = Category{ID: 0, Name: "Other"}
	CategoryOther_Misc         = Category{ID: 10, Name: "Other/Misc"}
	CategoryOther_Hashed       = Category{ID: 20, Name: "Other/Hashed"}
	CategoryConsole            = Category{ID: 1000, Name: "Console"}
	CategoryConsole_NDS        = Category{ID: 1010, Name: "Console/NDS"}
	CategoryConsole_PSP        = Category{ID: 1020, Name: "Console/PSP"}
	CategoryConsole_Wii        = Category{ID: 1030, Name: "Console/Wii"}
	CategoryConsole_XBOX       = Category{ID: 1040, Name: "Console/Xbox"}
	CategoryConsole_XBOX360    = Category{ID: 1050, Name: "Console/Xbox360"}
	CategoryConsole_WiiwareVC  = Category{ID: 1060, Name: "Console/Wiiware/V"}
	CategoryConsole_XBOX360DLC = Category{ID: 1070, Name: "Console/Xbox360"}
	CategoryConsole_PS3        = Category{ID: 1080, Name: "Console/PS3"}
	CategoryConsole_Other      = Category{ID: 1999, Name: "Console/Other"}
	CategoryConsole_3DS        = Category{ID: 1110, Name: "Console/3DS"}
	CategoryConsole_PSVita     = Category{ID: 1120, Name: "Console/PS Vita"}
	CategoryConsole_WiiU       = Category{ID: 1130, Name: "Console/WiiU"}
	CategoryConsole_XBOXOne    = Category{ID: 1140, Name: "Console/XboxOne"}
	CategoryConsole_PS4        = Category{ID: 1180, Name: "Console/PS4"}
	CategoryMovies             = Category{ID: 2000, Name: "Movies"}
	CategoryMovies_Foreign     = Category{ID: 2010, Name: "Movies/Foreign"}
	CategoryMovies_Other       = Category{ID: 2020, Name: "Movies/Other"}
	CategoryMovies_SD          = Category{ID: 2030, Name: "Movies/SD"}
	CategoryMovies_HD          = Category{ID: 2040, Name: "Movies/HD"}
	CategoryMovies_3D          = Category{ID: 2050, Name: "Movies/3D"}
	CategoryMovies_BluRay      = Category{ID: 2060, Name: "Movies/BluRay"}
	CategoryMovies_DVD         = Category{ID: 2070, Name: "Movies/DVD"}
	CategoryMovies_WEBDL       = Category{ID: 2080, Name: "Movies/WEBDL"}
	CategoryAudio              = Category{ID: 3000, Name: "Audio"}
	CategoryAudio_MP3          = Category{ID: 3010, Name: "Audio/MP3"}
	CategoryAudio_Video        = Category{ID: 3020, Name: "Audio/Video"}
	CategoryAudio_Audiobook    = Category{ID: 3030, Name: "Audio/Audiobook"}
	CategoryAudio_Lossless     = Category{ID: 3040, Name: "Audio/Lossless"}
	CategoryAudio_Other        = Category{ID: 3999, Name: "Audio/Other"}
	CategoryAudio_Foreign      = Category{ID: 3060, Name: "Audio/Foreign"}
	CategoryPC                 = Category{ID: 4000, Name: "PC"}
	CategoryPC_0day            = Category{ID: 4010, Name: "PC/0day"}
	CategoryPC_ISO             = Category{ID: 4020, Name: "PC/ISO"}
	CategoryPC_Mac             = Category{ID: 4030, Name: "PC/Mac"}
	CategoryPC_PhoneOther      = Category{ID: 4040, Name: "PC/Phone-Other"}
	CategoryPC_Games           = Category{ID: 4050, Name: "PC/Games"}
	CategoryPC_PhoneIOS        = Category{ID: 4060, Name: "PC/Phone-IOS"}
	CategoryPC_PhoneAndroid    = Category{ID: 4070, Name: "PC/Phone-Android"}
	CategoryTV                 = Category{ID: 5000, Name: "TV"}
	CategoryTV_WEBDL           = Category{ID: 5010, Name: "TV/WEB-DL"}
	CategoryTV_FOREIGN         = Category{ID: 5020, Name: "TV/Foreign"}
	CategoryTV_SD              = Category{ID: 5030, Name: "TV/SD"}
	CategoryTV_HD              = Category{ID: 5040, Name: "TV/HD"}
	CategoryTV_Other           = Category{ID: 5999, Name: "TV/Other"}
	CategoryTV_Sport           = Category{ID: 5060, Name: "TV/Sport"}
	CategoryTV_Anime           = Category{ID: 5070, Name: "TV/Anime"}
	CategoryTV_Documentary     = Category{ID: 5080, Name: "TV/Documentary"}
	CategoryXXX                = Category{ID: 6000, Name: "XXX"}
	CategoryXXX_DVD            = Category{ID: 6010, Name: "XXX/DVD"}
	CategoryXXX_WMV            = Category{ID: 6020, Name: "XXX/WMV"}
	CategoryXXX_XviD           = Category{ID: 6030, Name: "XXX/XviD"}
	CategoryXXX_x264           = Category{ID: 6040, Name: "XXX/x264"}
	CategoryXXX_Other          = Category{ID: 6999, Name: "XXX/Other"}
	CategoryXXX_Imageset       = Category{ID: 6060, Name: "XXX/Imageset"}
	CategoryXXX_Packs          = Category{ID: 6070, Name: "XXX/Packs"}
	CategoryBooks              = Category{ID: 7000, Name: "Books"}
	CategoryBooks_Magazines    = Category{ID: 7010, Name: "Books/Magazines"}
	CategoryBooks_Ebook        = Category{ID: 7020, Name: "Books/Ebook"}
	CategoryBooks_Comics       = Category{ID: 7030, Name: "Books/Comics"}
	CategoryBooks_Technical    = Category{ID: 7040, Name: "Books/Technical"}
	CategoryBooks_Foreign      = Category{ID: 7060, Name: "Books/Foreign"}
	CategoryBooks_Unknown      = Category{ID: 7999, Name: "Books/Unknown"}
)

var AllCategories = Categories{
	CategoryOther,
	CategoryOther_Misc,
	CategoryOther_Hashed,
	CategoryConsole,
	CategoryConsole_NDS,
	CategoryConsole_PSP,
	CategoryConsole_Wii,
	CategoryConsole_XBOX,
	CategoryConsole_XBOX360,
	CategoryConsole_WiiwareVC,
	CategoryConsole_XBOX360DLC,
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
	CategoryMovies_3D,
	CategoryMovies_BluRay,
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
	CategoryPC_PhoneOther,
	CategoryPC_Games,
	CategoryPC_PhoneIOS,
	CategoryPC_PhoneAndroid,
	CategoryTV,
	CategoryTV_WEBDL,
	CategoryTV_FOREIGN,
	CategoryTV_SD,
	CategoryTV_HD,
	CategoryTV_Other,
	CategoryTV_Sport,
	CategoryTV_Anime,
	CategoryTV_Documentary,
	CategoryXXX,
	CategoryXXX_DVD,
	CategoryXXX_WMV,
	CategoryXXX_XviD,
	CategoryXXX_x264,
	CategoryXXX_Other,
	CategoryXXX_Imageset,
	CategoryXXX_Packs,
	CategoryBooks,
	CategoryBooks_Magazines,
	CategoryBooks_Ebook,
	CategoryBooks_Comics,
	CategoryBooks_Technical,
	CategoryBooks_Foreign,
	CategoryBooks_Unknown,
}

func ParentCategory(c Category) Category {
	switch {
	case c.ID < 1000:
		return CategoryOther
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
	}
	return CategoryOther
}

type Categories = znab.Categories
