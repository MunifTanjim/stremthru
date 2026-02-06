package newznabcache

import (
	"net/http"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/logger"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
)

type cachedSearcher struct {
	cache cache.Cache[[]newznab_client.Newz]
}

func (cs *cachedSearcher) Do(idxr newznab_client.Indexer, query url.Values, headers http.Header, log *logger.Logger) ([]newznab_client.Newz, error) {
	apiKey := query.Get("apikey")
	query.Del("apikey")
	encQuery := query.Encode()
	if apiKey != "" {
		query.Set("apikey", apiKey)
	}

	cacheKey := idxr.GetId() + ":" + encQuery

	var items []newznab_client.Newz
	var err error

	if cs.cache.Get(cacheKey, &items) {
		if log != nil {
			log.Debug("indexer search cache hit", "indexer", idxr.GetId(), "query", encQuery, "count", len(items))
		}
	} else {
		start := time.Now()
		items, err = idxr.Search(query, headers)
		if err == nil {
			if log != nil {
				log.Debug("indexer search completed", "indexer", idxr.GetId(), "query", encQuery, "duration", time.Since(start).String(), "count", len(items))
			}
			cs.cache.Add(cacheKey, items)
		} else {
			if log != nil {
				log.Error("indexer search failed", "error", err, "indexer", idxr.GetId(), "query", encQuery, "duration", time.Since(start).String())
			}
		}
	}

	return items, err
}

var Search = cachedSearcher{
	cache: cache.NewCache[[]newznab_client.Newz](&cache.CacheConfig{
		Name:     "newznab:search",
		Lifetime: 3 * time.Hour,
		MaxSize:  512,
		Persist:  true,
	}),
}
