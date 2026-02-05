package newznab_indexer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
)

var indexerCache = cache.NewLRUCache[newznab_client.Indexer](&cache.CacheConfig{
	Name:     "newznab:indexer",
	Lifetime: 3 * time.Hour,
})

func (idxr *NewznabIndexer) GetClient() (newznab_client.Indexer, error) {
	switch idxr.Type {
	case IndexerTypeGeneric:
		apiKey, err := idxr.GetAPIKey()
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt api key: %w", err)
		}

		cacheKey := strconv.FormatInt(idxr.Id, 10)
		var client newznab_client.Indexer
		if !indexerCache.Get(cacheKey, &client) {
			client = newznab_client.NewClient(&newznab_client.ClientConfig{
				BaseURL: idxr.URL,
				APIKey:  apiKey,
			})

			err := indexerCache.Add(cacheKey, client)
			if err != nil {
				return nil, err
			}
		}

		return client, nil

	default:
		return nil, fmt.Errorf("invalid indexer type: %s", idxr.Type)
	}
}
