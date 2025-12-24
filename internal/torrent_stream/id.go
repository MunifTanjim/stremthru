package torrent_stream

import (
	"errors"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/cache"
)

func CleanStremId(sid string) string {
	if strings.HasPrefix(sid, "tt") {
		sid, _, _ = strings.Cut(sid, ":")
		return sid
	}
	if sid, found := strings.CutPrefix(sid, "anidb:"); found {
		sid, _, _ = strings.Cut(sid, ":")
		if sid != "" {
			return "anidb:" + sid
		}
	}
	return ""
}

// imdb or anidb
type NormalizedStremId struct {
	IsAnime bool   // when `true`, `Id` is AniDB id.
	Id      string // imdb or anidb
	Episode string
	Season  string
}

func (nsid NormalizedStremId) IsSeries() bool {
	return nsid.Season != "" || nsid.Episode != ""
}

func (nsid NormalizedStremId) ToClean() string {
	if nsid.IsAnime && nsid.Id != "" {
		return "anidb:" + nsid.Id
	}
	return nsid.Id
}

func (nsid NormalizedStremId) String() string {
	id := nsid.ToClean()
	if id == "" {
		return id
	}
	if nsid.IsAnime {
		if nsid.Episode != "" {
			id += ":" + nsid.Episode
		}
		return id
	}
	if nsid.Season != "" {
		id += ":" + nsid.Season
		if nsid.Episode != "" {
			id += ":" + nsid.Episode
		}
	}
	return id
}

var normalizedStremIdCache = cache.NewLRUCache[NormalizedStremId](&cache.CacheConfig{
	Lifetime: 60 * time.Second,
	Name:     "normalized_strem_id",
})

var ErrUnsupportedStremId = errors.New("unsupported strem id")

// to imdb or anidb
func NormalizeStreamId(sid string) (*NormalizedStremId, error) {
	result := NormalizedStremId{}

	if normalizedStremIdCache.Get(sid, &result) {
		return &result, nil
	}

	if strings.HasPrefix(sid, "tt") {
		imdbId, seasonEpisode, ok := strings.Cut(sid, ":")
		result.Id = imdbId
		if ok {
			season, episode, ok := strings.Cut(seasonEpisode, ":")
			result.Season = season
			if ok {
				result.Episode = episode
			}
		}
	} else if idEpisode, ok := strings.CutPrefix(sid, "anidb:"); ok {
		anidbId, episode, _ := strings.Cut(idEpisode, ":")
		season, err := anime.GetAniDBSeasonById(anidbId)
		if err != nil {
			return nil, err
		}
		result.IsAnime = true
		result.Id = anidbId
		result.Season = season
		result.Episode = episode
	} else if idEpisode, ok := strings.CutPrefix(sid, "kitsu:"); ok {
		kitsuId, episode, _ := strings.Cut(idEpisode, ":")
		anidbId, season, err := anime.GetAniDBIdByKitsuId(kitsuId)
		if err != nil {
			return nil, err
		}
		result.IsAnime = true
		result.Id = anidbId
		result.Season = season
		result.Episode = episode
	} else if idEpisode, ok := strings.CutPrefix(sid, "mal:"); ok {
		malId, episode, _ := strings.Cut(idEpisode, ":")
		anidbId, season, err := anime.GetAniDBIdByMALId(malId)
		if err != nil {
			return nil, err
		}
		result.IsAnime = true
		result.Id = anidbId
		result.Season = season
		result.Episode = episode
	} else {
		return nil, ErrUnsupportedStremId
	}

	if err := normalizedStremIdCache.Add(sid, result); err != nil {
		return nil, err
	}

	return &result, nil
}
