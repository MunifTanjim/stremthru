package znab

type SearchParam = string

const (
	SearchParamT        SearchParam = "t"
	SearchParamAPIKey   SearchParam = "apikey"
	SearchParamCat      SearchParam = "cat"
	SearchParamAttrs    SearchParam = "attrs"
	SearchParamExtended SearchParam = "extended"
	SearchParamOffset   SearchParam = "offset"
	SearchParamLimit    SearchParam = "limit"
	SearchParamQ        SearchParam = "q"
	SearchParamEp       SearchParam = "ep"
	SearchParamSeason   SearchParam = "season"
	SearchParamYear     SearchParam = "year"
	SearchParamIMDBId   SearchParam = "imdbid"
	SearchParamTVDBId   SearchParam = "tvdbid"
	SearchParamTVMazeId SearchParam = "tvmazeid"
	SearchParamTraktId  SearchParam = "traktid"
	SearchParamRageId   SearchParam = "rid" // TVRage ID
)
