package meta

import (
	"errors"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	meta_type "github.com/MunifTanjim/stremthru/internal/meta/type"
)

type IdType = meta_type.IdType

const (
	IdTypeMovie   = meta_type.IdTypeMovie
	IdTypeShow    = meta_type.IdTypeShow
	IdTypeUnknown = meta_type.IdTypeUnknown
)

type IdProvider = meta_type.IdProvider

const (
	IdProviderIMDB        = meta_type.IdProviderIMDB
	IdProviderTMDB        = meta_type.IdProviderTMDB
	IdProviderTVDB        = meta_type.IdProviderTVDB
	IdProviderTVMaze      = meta_type.IdProviderTVMaze
	IdProviderTrakt       = meta_type.IdProviderTrakt
	IdProviderLetterboxd  = meta_type.IdProviderLetterboxd
	IdProviderAniDB       = meta_type.IdProviderAniDB
	IdProviderAniList     = meta_type.IdProviderAniList
	IdProviderAniSearch   = meta_type.IdProviderAniSearch
	IdProviderAnimePlanet = meta_type.IdProviderAnimePlanet
	IdProviderKitsu       = meta_type.IdProviderKitsu
	IdProviderLiveChart   = meta_type.IdProviderLiveChart
	IdProviderMAL         = meta_type.IdProviderMAL
	IdProviderNotifyMoe   = meta_type.IdProviderNotifyMoe
)

type IdMapAnime = meta_type.IdMapAnime
type IdMap = meta_type.IdMap

func ParseId(idStr string) (provider IdProvider, id string) {
	if strings.HasPrefix(idStr, "tt") {
		return IdProviderIMDB, idStr
	}
	if traktId, ok := strings.CutPrefix(idStr, "trakt:"); ok {
		return IdProviderTrakt, traktId
	}
	if tvdbId, ok := strings.CutPrefix(idStr, "tvdb:"); ok {
		return IdProviderTVDB, tvdbId
	}
	return "", ""
}

var ErrorUnsupportedId = errors.New("unsupported id")
var ErrorUnsupportedIdAnchor = errors.New("unsupported id anchor")

var idMapCache = cache.NewCache[IdMap](&cache.CacheConfig{
	Lifetime: 3 * time.Hour,
	Name:     "meta:id-map",
	MaxSize:  2048,
})

func GetIdMap(idType IdType, idStr string) (*IdMap, error) {
	idProvider, id := ParseId(idStr)

	idMap := IdMap{IMDB: id}

	cacheKey := meta_type.GetIdProviderCacheKey(idProvider, idType, id)
	if !idMapCache.Get(cacheKey, &idMap) {
		switch idProvider {
		case IdProviderIMDB:
			idm, err := imdb_title.GetIdMapByIMDBId(id)
			if err != nil || idm == nil {
				return &idMap, err
			}
			idMap.Type = IdType(idm.Type.ToSimple())
			idMap.IMDB = id
			idMap.TMDB = idm.TMDBId
			idMap.TVDB = idm.TVDBId
			idMap.Trakt = idm.TraktId
			idMap.Letterboxd = idm.LetterboxdId
		case IdProviderTVDB:
			idm, err := imdb_title.GetIdMapByTVDBId(imdb_title.IMDBTitleSimpleType(idType), id)
			if err != nil || idm == nil {
				return &idMap, err
			}
			idMap.Type = IdType(idm.Type.ToSimple())
			idMap.IMDB = idm.IMDBId
			idMap.TMDB = idm.TMDBId
			idMap.TVDB = id
			idMap.Trakt = idm.TraktId
			idMap.Letterboxd = idm.LetterboxdId
		case IdProviderTrakt:
			idMaps, err := imdb_title.GetIdMapsByTraktIds(imdb_title.IMDBTitleSimpleType(idType), []string{id})
			if err != nil {
				return &idMap, err
			}
			idm := idMaps[id]
			idMap.Type = IdType(idm.Type.ToSimple())
			idMap.IMDB = idm.IMDBId
			idMap.TMDB = idm.TMDBId
			idMap.TVDB = idm.TVDBId
			idMap.Trakt = id
			idMap.Letterboxd = idm.LetterboxdId
		default:
			return nil, ErrorUnsupportedId
		}

		if err := idMapCache.Add(cacheKey, idMap); err != nil {
			return nil, err
		}
	}

	return &idMap, nil
}

func SetIdMapsInTrx(tx db.Executor, idMaps []IdMap, anchor IdProvider) error {
	if anchor != IdProviderIMDB {
		return ErrorUnsupportedIdAnchor
	}

	cacheKeys := make([]string, 0, len(idMaps))
	imdbMapItems := []imdb_title.BulkRecordMappingInputItem{}
	for _, idMap := range idMaps {
		if idMap.IMDB == "" {
			continue
		}
		cacheKeys = append(cacheKeys, anchor.GetCacheKey(idMap))
		imdbMap := imdb_title.BulkRecordMappingInputItem{
			IMDBId:       idMap.IMDB,
			TMDBId:       idMap.TMDB,
			TVDBId:       idMap.TVDB,
			TraktId:      idMap.Trakt,
			LetterboxdId: idMap.Letterboxd,
		}
		if idMap.Anime != nil && idMap.Anime.MAL != "" {
			imdbMap.MALId = idMap.Anime.MAL
		}
		imdbMapItems = append(imdbMapItems, imdbMap)
	}

	for _, cacheKey := range cacheKeys {
		idMapCache.Remove(cacheKey)
	}

	return imdb_title.BulkRecordMapping(tx, imdbMapItems)
}

func SetIdMaps(idMaps []IdMap, anchor IdProvider) error {
	return SetIdMapsInTrx(db.GetDB(), idMaps, anchor)
}
