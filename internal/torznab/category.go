package torznab

import "fmt"

type Category struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type jsonCategory struct {
	Attributes Category `json:"@attributes"`
}

func (c Category) String() string {
	return fmt.Sprintf("%s[%d]", c.Name, c.ID)
}

const (
	CustomCategoryOffset = 100000
)

// Categories from the Newznab spec
// https://github.com/nZEDb/nZEDb/blob/main/docs/newznab_api_specification.txt#L627
var (
	CategoryOther              = Category{0, "Other"}
	CategoryOther_Misc         = Category{10, "Other/Misc"}
	CategoryOther_Hashed       = Category{20, "Other/Hashed"}
	CategoryConsole            = Category{1000, "Console"}
	CategoryConsole_NDS        = Category{1010, "Console/NDS"}
	CategoryConsole_PSP        = Category{1020, "Console/PSP"}
	CategoryConsole_Wii        = Category{1030, "Console/Wii"}
	CategoryConsole_XBOX       = Category{1040, "Console/Xbox"}
	CategoryConsole_XBOX360    = Category{1050, "Console/Xbox360"}
	CategoryConsole_WiiwareVC  = Category{1060, "Console/Wiiware/V"}
	CategoryConsole_XBOX360DLC = Category{1070, "Console/Xbox360"}
	CategoryConsole_PS3        = Category{1080, "Console/PS3"}
	CategoryConsole_Other      = Category{1999, "Console/Other"}
	CategoryConsole_3DS        = Category{1110, "Console/3DS"}
	CategoryConsole_PSVita     = Category{1120, "Console/PS Vita"}
	CategoryConsole_WiiU       = Category{1130, "Console/WiiU"}
	CategoryConsole_XBOXOne    = Category{1140, "Console/XboxOne"}
	CategoryConsole_PS4        = Category{1180, "Console/PS4"}
	CategoryMovies             = Category{2000, "Movies"}
	CategoryMovies_Foreign     = Category{2010, "Movies/Foreign"}
	CategoryMovies_Other       = Category{2020, "Movies/Other"}
	CategoryMovies_SD          = Category{2030, "Movies/SD"}
	CategoryMovies_HD          = Category{2040, "Movies/HD"}
	CategoryMovies_3D          = Category{2050, "Movies/3D"}
	CategoryMovies_BluRay      = Category{2060, "Movies/BluRay"}
	CategoryMovies_DVD         = Category{2070, "Movies/DVD"}
	CategoryMovies_WEBDL       = Category{2080, "Movies/WEBDL"}
	CategoryAudio              = Category{3000, "Audio"}
	CategoryAudio_MP3          = Category{3010, "Audio/MP3"}
	CategoryAudio_Video        = Category{3020, "Audio/Video"}
	CategoryAudio_Audiobook    = Category{3030, "Audio/Audiobook"}
	CategoryAudio_Lossless     = Category{3040, "Audio/Lossless"}
	CategoryAudio_Other        = Category{3999, "Audio/Other"}
	CategoryAudio_Foreign      = Category{3060, "Audio/Foreign"}
	CategoryPC                 = Category{4000, "PC"}
	CategoryPC_0day            = Category{4010, "PC/0day"}
	CategoryPC_ISO             = Category{4020, "PC/ISO"}
	CategoryPC_Mac             = Category{4030, "PC/Mac"}
	CategoryPC_PhoneOther      = Category{4040, "PC/Phone-Other"}
	CategoryPC_Games           = Category{4050, "PC/Games"}
	CategoryPC_PhoneIOS        = Category{4060, "PC/Phone-IOS"}
	CategoryPC_PhoneAndroid    = Category{4070, "PC/Phone-Android"}
	CategoryTV                 = Category{5000, "TV"}
	CategoryTV_WEBDL           = Category{5010, "TV/WEB-DL"}
	CategoryTV_FOREIGN         = Category{5020, "TV/Foreign"}
	CategoryTV_SD              = Category{5030, "TV/SD"}
	CategoryTV_HD              = Category{5040, "TV/HD"}
	CategoryTV_Other           = Category{5999, "TV/Other"}
	CategoryTV_Sport           = Category{5060, "TV/Sport"}
	CategoryTV_Anime           = Category{5070, "TV/Anime"}
	CategoryTV_Documentary     = Category{5080, "TV/Documentary"}
	CategoryXXX                = Category{6000, "XXX"}
	CategoryXXX_DVD            = Category{6010, "XXX/DVD"}
	CategoryXXX_WMV            = Category{6020, "XXX/WMV"}
	CategoryXXX_XviD           = Category{6030, "XXX/XviD"}
	CategoryXXX_x264           = Category{6040, "XXX/x264"}
	CategoryXXX_Other          = Category{6999, "XXX/Other"}
	CategoryXXX_Imageset       = Category{6060, "XXX/Imageset"}
	CategoryXXX_Packs          = Category{6070, "XXX/Packs"}
	CategoryBooks              = Category{7000, "Books"}
	CategoryBooks_Magazines    = Category{7010, "Books/Magazines"}
	CategoryBooks_Ebook        = Category{7020, "Books/Ebook"}
	CategoryBooks_Comics       = Category{7030, "Books/Comics"}
	CategoryBooks_Technical    = Category{7040, "Books/Technical"}
	CategoryBooks_Foreign      = Category{7060, "Books/Foreign"}
	CategoryBooks_Unknown      = Category{7999, "Books/Unknown"}
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

type Categories []Category

func (slice Categories) Subset(ids ...int) Categories {
	cats := Categories{}

	for _, cat := range AllCategories {
		for _, id := range ids {
			if cat.ID == id {
				cats = append(cats, cat)
			}
		}
	}

	return cats
}

func (slice Categories) Len() int {
	return len(slice)
}

func (slice Categories) Less(i, j int) bool {
	return slice[i].ID < slice[j].ID
}

func (slice Categories) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
