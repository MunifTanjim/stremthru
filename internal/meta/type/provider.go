package meta_type

type Provider string

const (
	ProviderLetterboxd Provider = "letterboxd"
	ProviderTMDB       Provider = "tmdb"
	ProviderTVDB       Provider = "tvdb"
)

type IdProvider string

const (
	IdProviderIMDB        IdProvider = "imdb"
	IdProviderTMDB        IdProvider = IdProvider(ProviderTMDB)
	IdProviderTVDB        IdProvider = IdProvider(ProviderTVDB)
	IdProviderTVMaze      IdProvider = "tvmaze"
	IdProviderTrakt       IdProvider = "trakt"
	IdProviderLetterboxd  IdProvider = "lboxd"
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

func GetIdProviderCacheKey(idProvider IdProvider, idType IdType, id string) string {
	switch idProvider {
	case IdProviderIMDB:
		return id
	case IdProviderTrakt, IdProviderTVDB:
		return string(idProvider) + ":" + string(idType) + ":" + id
	default:
		panic("unsupported id provider: " + string(idProvider))
	}
}

func (ip IdProvider) GetCacheKey(idMap IdMap) string {
	return GetIdProviderCacheKey(ip, idMap.Type, idMap.IMDB)
}
