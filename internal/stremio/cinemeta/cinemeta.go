package cinemeta

import (
	"net/url"
	"slices"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/stremio"
	"golang.org/x/sync/singleflight"
)

var client = stremio_addon.NewClient(&stremio_addon.ClientConfig{})
var baseUrl, _ = url.Parse("https://v3-cinemeta.strem.io/")
var metaCache = cache.NewCache[stremio.Meta](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "stremio:cinemeta:meta",
})
var fetchMetaGroup singleflight.Group

func FetchMeta(sType, imdbId string) (stremio.Meta, error) {
	var meta stremio.Meta
	cacheKey := sType + ":" + imdbId
	if !metaCache.Get(cacheKey, &meta) {
		m, err, _ := fetchMetaGroup.Do(cacheKey, func() (any, error) {
			r, err := client.FetchMeta(&stremio_addon.FetchMetaParams{
				BaseURL: baseUrl,
				Type:    sType,
				Id:      imdbId + ".json",
			})
			return r.Data.Meta, err
		})
		if err != nil {
			return meta, err
		}
		meta = m.(stremio.Meta)
		slices.SortFunc(meta.Videos, func(a, b stremio.MetaVideo) int {
			if a.Season != b.Season {
				return int(a.Season) - int(b.Season)
			}
			return int(a.Episode) - int(b.Episode)
		})
		metaCache.Add(cacheKey, meta)
	}
	return meta, nil
}
