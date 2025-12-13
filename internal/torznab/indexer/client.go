package torznab_indexer

import (
	"errors"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

var jackettCache = cache.NewLRUCache[*jackett.Client](&cache.CacheConfig{
	Lifetime: 3 * time.Hour,
	Name:     "torznab:indexer:jackett",
})

func (tidxr TorznabIndexer) GetClient() (torznab_client.Indexer, error) {
	switch tidxr.Type {
	case IndexerTypeJackett:
		apiKey, err := tidxr.GetAPIKey()
		if err != nil {
			return nil, err
		}

		u := jackett.TorznabURL(tidxr.URL)
		if err := u.Parse(); err != nil {
			return nil, err
		}

		var client *jackett.Client
		if !jackettCache.Get(tidxr.Id, &client) {
			client = jackett.NewClient(&jackett.ClientConfig{
				BaseURL: u.BaseURL,
				APIKey:  apiKey,
			})
			err := jackettCache.Add(tidxr.Id, client)
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
