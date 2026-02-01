package stremio_userdata

import (
	"fmt"
	"slices"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
)

type NewzIndexerType string

const (
	NewzIndexerTypeGeneric NewzIndexerType = "generic"
)

type NewzIndexer struct {
	Type   NewzIndexerType `json:"t"`
	Name   string          `json:"n"`
	URL    string          `json:"u"`
	APIKey string          `json:"ak,omitempty"`
}

func (i NewzIndexer) Validate() (string, error) {
	if i.Type == "" {
		return "type", fmt.Errorf("indexer type is required")
	}
	if i.Name == "" {
		return "name", fmt.Errorf("indexer name is required")
	}
	if i.URL == "" {
		return "url", fmt.Errorf("indexer url is required")
	}
	return "", nil
}

type UserDataNewzIndexers struct {
	Indexers []NewzIndexer `json:"indexers"`
}

func (ud UserDataNewzIndexers) HasRequiredValues() bool {
	indexerCount := len(ud.Indexers)
	if indexerCount == 0 {
		return false
	}
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		if _, err := indexer.Validate(); err != nil {
			return false
		}
	}
	return true
}

func (ud UserDataNewzIndexers) StripSecrets() UserDataNewzIndexers {
	ud.Indexers = slices.Clone(ud.Indexers)
	for i := range ud.Indexers {
		s := &ud.Indexers[i]
		s.APIKey = ""
	}
	return ud
}

var newznabClientCache = cache.NewLRUCache[*newznab_client.Client](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "stremio:userdata:newz_indexers",
})

func (ud *UserDataNewzIndexers) Prepare() ([]newznab_client.Indexer, error) {
	clients := make([]newznab_client.Indexer, 0, len(ud.Indexers))
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]

		baseURL := indexer.URL
		apiKey := indexer.APIKey

		switch indexer.Type {
		case NewzIndexerTypeGeneric:
			key := baseURL + ":" + apiKey
			var client *newznab_client.Client
			if !newznabClientCache.Get(key, &client) {
				client = newznab_client.NewClient(&newznab_client.ClientConfig{
					BaseURL: baseURL,
					APIKey:  apiKey,
				})
				err := newznabClientCache.Add(key, client)
				if err != nil {
					return clients, err
				}
			}
			clients = append(clients, client)

		default:
			return clients, fmt.Errorf("unsupported newz indexer type: %s", indexer.Type)
		}
	}
	return clients, nil
}
