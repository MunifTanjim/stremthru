package meta

import (
	"errors"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
)

type IdType string

const (
	IdTypeMovie   IdType = "movie"
	IdTypeShow    IdType = "show"
	IdTypeUnknown IdType = ""
)

func (it IdType) IsValid() bool {
	return it == IdTypeUnknown || it == IdTypeMovie || it == IdTypeShow
}

type IdMapAnime struct {
	AniDB       string `json:"anidb,omitempty"`
	AniList     string `json:"anilist,omitempty"`
	AniSearch   string `json:"anisearch,omitempty"`
	AnimePlanet string `json:"animeplanet,omitempty"`
	Kitsu       string `json:"kitsu,omitempty"`
	LiveChart   string `json:"livechart,omitempty"`
	MAL         string `json:"mal,omitempty"`
	NotifyMoe   string `json:"notifymoe,omitempty"`
}

type IdMap struct {
	Type   IdType      `json:"type"`
	IMDB   string      `json:"imdb,omitempty"`
	TMDB   string      `json:"tmdb,omitempty"`
	TVDB   string      `json:"tvdb,omitempty"`
	TVMaze string      `json:"tvmaze,omitempty"`
	Trakt  string      `json:"trakt,omitempty"`
	Anime  *IdMapAnime `json:"anime,omitempty"`
}

type IdProvider string

const (
	IdProviderIMDB        IdProvider = "imdb"
	IdProviderTMDB        IdProvider = "tmdb"
	IdProviderTVDB        IdProvider = "tvdb"
	IdProviderTVMaze      IdProvider = "tvmaze"
	IdProviderTrakt       IdProvider = "trakt"
	IdProviderAniDB       IdProvider = "anidb"
	IdProviderAniList     IdProvider = "anilist"
	IdProviderAniSearch   IdProvider = "anisearch"
	IdProviderAnimePlanet IdProvider = "animeplanet"
	IdProviderKitsu       IdProvider = "kitsu"
	IdProviderLiveChart   IdProvider = "livechart"
	IdProviderMAL         IdProvider = "mal"
	IdProviderNotifyMoe   IdProvider = "notifymoe"
)

func (ip IdProvider) IsAnime() bool {
	return ip == IdProviderAniDB ||
		ip == IdProviderAniList ||
		ip == IdProviderAniSearch ||
		ip == IdProviderAnimePlanet ||
		ip == IdProviderKitsu ||
		ip == IdProviderLiveChart ||
		ip == IdProviderMAL ||
		ip == IdProviderNotifyMoe
}

func getCacheKey(idProvider IdProvider, idType IdType, id string) string {
	switch idProvider {
	case IdProviderIMDB:
		return id
	case IdProviderTVDB:
		return string(idProvider) + ":" + string(idType) + ":" + id
	default:
		panic("unsupported id provider: " + string(idProvider))
	}
}

func (ip IdProvider) GetCacheKey(idMap IdMap) string {
	return getCacheKey(ip, idMap.Type, idMap.IMDB)
}

func ParseId(idStr string) (provider IdProvider, id string) {
	if strings.HasPrefix(idStr, "tt") {
		return IdProviderIMDB, idStr
	}
	if tvdbId, ok := strings.CutPrefix(idStr, "tvdb:"); ok {
		return IdProviderTVDB, tvdbId
	}
	return "", ""
}

var ErrorUnsupportedId = errors.New("unsupported id")
var ErrorUnsupportedIdAnchor = errors.New("unsupported id anchor")

var idMapCache = cache.NewCache[IdMap](&cache.CacheConfig{
	Lifetime:      3 * time.Hour,
	Name:          "meta:id-map",
	LocalCapacity: 2048,
})

func GetIdMap(idType IdType, idStr string) (*IdMap, error) {
	idProvider, id := ParseId(idStr)

	idMap := IdMap{IMDB: id}

	cacheKey := getCacheKey(idProvider, idType, id)
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
		case IdProviderTVDB:
			idm, err := imdb_title.GetIdMapByTVDBId(id)
			if err != nil || idm == nil {
				return &idMap, err
			}
			idMap.Type = IdType(idm.Type.ToSimple())
			idMap.IMDB = idm.IMDBId
			idMap.TMDB = idm.TMDBId
			idMap.TVDB = id
			idMap.Trakt = idm.TraktId
		default:
			return nil, ErrorUnsupportedId
		}

		if err := idMapCache.Add(cacheKey, idMap); err != nil {
			return nil, err
		}
	}

	return &idMap, nil
}

func SetIdMaps(idMaps []IdMap, anchor IdProvider) error {
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
			IMDBId:  idMap.IMDB,
			TMDBId:  idMap.TMDB,
			TVDBId:  idMap.TVDB,
			TraktId: idMap.Trakt,
		}
		if idMap.Anime != nil && idMap.Anime.MAL != "" {
			imdbMap.MALId = idMap.Anime.MAL
		}
		imdbMapItems = append(imdbMapItems, imdbMap)
	}

	for _, cacheKey := range cacheKeys {
		idMapCache.Remove(cacheKey)
	}

	return imdb_title.BulkRecordMapping(imdbMapItems)
}
