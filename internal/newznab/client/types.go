package newznab_client

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

type ChannelItemAttrName = znab.ChannelItemAttrName

type SearchParam = znab.SearchParam

const (
	SearchParamT        = znab.SearchParamT
	SearchParamAPIKey   = znab.SearchParamAPIKey
	SearchParamCat      = znab.SearchParamCat
	SearchParamAttrs    = znab.SearchParamAttrs
	SearchParamExtended = znab.SearchParamExtended
	SearchParamOffset   = znab.SearchParamOffset
	SearchParamLimit    = znab.SearchParamLimit
	SearchParamQ        = znab.SearchParamQ
	SearchParamEp       = znab.SearchParamEp
	SearchParamSeason   = znab.SearchParamSeason
	SearchParamYear     = znab.SearchParamYear
	SearchParamIMDBId   = znab.SearchParamIMDBId
	SearchParamTVDBId   = znab.SearchParamTVDBId
	SearchParamTVMazeId = znab.SearchParamTVMazeId
	SearchParamTraktId  = znab.SearchParamTraktId
	SearchParamRageId   = znab.SearchParamRageId
)
