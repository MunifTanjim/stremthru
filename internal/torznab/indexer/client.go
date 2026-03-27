package torznab_indexer

import (
	"errors"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/generic"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var genericCache = cache.NewLRUCache[*generic.TorznabClient](&cache.CacheConfig{
	Lifetime: 3 * time.Hour,
	Name:     "torznab:indexer:generic",
})

var jackettCache = cache.NewLRUCache[*jackett.Client](&cache.CacheConfig{
	Lifetime: 3 * time.Hour,
	Name:     "torznab:indexer:jackett",
})

func (tidxr TorznabIndexer) GetClient() (torznab_client.Indexer, error) {
	switch tidxr.Type {
	case IndexerTypeGeneric:
		apiKey, err := tidxr.GetAPIKey()
		if err != nil {
			return nil, err
		}

		cacheKey := util.IntToString(tidxr.Id)
		var client *generic.TorznabClient
		if !genericCache.Get(cacheKey, &client) {
			client = generic.NewClient(&generic.TorznabClientConfig{
				BaseURL: tidxr.URL,
				APIKey:  apiKey,
				ID:      tidxr.Id,
				Name:    tidxr.Name,
			})
			err := genericCache.Add(cacheKey, client)
			if err != nil {
				return nil, err
			}
		}
		return client, nil
	case IndexerTypeJackett:
		apiKey, err := tidxr.GetAPIKey()
		if err != nil {
			return nil, err
		}

		u := jackett.TorznabURL(tidxr.URL)
		if err := u.Parse(); err != nil {
			return nil, err
		}

		cacheKey := strconv.FormatInt(tidxr.Id, 10)
		var client *jackett.Client
		if !jackettCache.Get(cacheKey, &client) {
			client = jackett.NewClient(&jackett.ClientConfig{
				BaseURL: u.BaseURL,
				APIKey:  apiKey,
			})
			err := jackettCache.Add(cacheKey, client)
			if err != nil {
				return nil, err
			}
		}
		c := client.GetTorznabClient(u.IndexerId)
		return c, nil
	default:
		return nil, errors.New("invalid indexer type: " + string(tidxr.Type))
	}
}
