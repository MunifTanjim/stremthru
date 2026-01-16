package torznab_client

import "github.com/MunifTanjim/stremthru/internal/znab"

type Caps = znab.Caps

type Function = znab.Function

const (
	FunctionCaps        = znab.FunctionCaps
	FunctionSearch      = znab.FunctionSearch
	FunctionSearchTV    = znab.FunctionSearchTV
	FunctionSearchMovie = znab.FunctionSearchMovie
	FunctionSearchMusic = znab.FunctionSearchMusic
	FunctionSearchBook  = znab.FunctionSearchBook
)

type SearchParam = znab.SearchParam

const (
	SearchParamT        SearchParam = znab.SearchParamT
	SearchParamAPIKey   SearchParam = znab.SearchParamAPIKey
	SearchParamCat      SearchParam = znab.SearchParamCat
	SearchParamAttrs    SearchParam = znab.SearchParamAttrs
	SearchParamExtended SearchParam = znab.SearchParamExtended
	SearchParamOffset   SearchParam = znab.SearchParamOffset
	SearchParamLimit    SearchParam = znab.SearchParamLimit
	SearchParamQ        SearchParam = znab.SearchParamQ
	SearchParamEp       SearchParam = znab.SearchParamEp
	SearchParamSeason   SearchParam = znab.SearchParamSeason
	SearchParamYear     SearchParam = znab.SearchParamYear
	SearchParamIMDBId   SearchParam = znab.SearchParamIMDBId
	SearchParamTVDBId   SearchParam = znab.SearchParamTVDBId
	SearchParamTVMazeId SearchParam = znab.SearchParamTVMazeId
	SearchParamTraktId  SearchParam = znab.SearchParamTraktId
)
