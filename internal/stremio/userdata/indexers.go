package stremio_userdata

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/generic"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

type IndexerName string

const (
	IndexerNameGeneric IndexerName = "generic"
	IndexerNameJackett IndexerName = "jackett"
)

type Indexer struct {
	Name   IndexerName `json:"n"`
	URL    string      `json:"u"`
	APIKey string      `json:"ak,omitempty"`
}

// type rawIndexer Indexer
//
// func (i Indexer) MarshalJSON() ([]byte, error) {
// 	i.Compress()
// 	return json.Marshal(rawIndexer(i))
// }
//
// func (i *Indexer) UnmarshalJSON(data []byte) error {
// 	ri := rawIndexer{}
// 	err := json.Unmarshal(data, &ri)
// 	if err != nil {
// 		return err
// 	}
// 	*i = Indexer(ri)
// 	i.Decompress()
// 	return nil
// }

func (i *Indexer) Decompress() {
	switch i.Name {
	case IndexerNameJackett:
		i.URL = jackett.TorznabURL(i.URL).Decode()
	}
}

func (i *Indexer) Compress() {
	switch i.Name {
	case IndexerNameJackett:
		i.URL = jackett.TorznabURL(i.URL).Encode()
	}
}

func (i Indexer) Validate() (string, error) {
	if i.Name == "" {
		return "name", fmt.Errorf("indexer name is required")
	}
	if i.URL == "" {
		return "url", fmt.Errorf("indexer url is required")
	}
	switch i.Name {
	case IndexerNameJackett:
		if err := jackett.TorznabURL(i.URL).Parse(); err != nil {
			return "url", fmt.Errorf("indexer url is invalid")
		}
	}
	return "", nil
}

type UserDataIndexers struct {
	Indexers []Indexer `json:"indexers"`
}

func (ud UserDataIndexers) HasRequiredValues() bool {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		if _, err := indexer.Validate(); err != nil {
			return false
		}
	}
	return true
}

func (ud UserDataIndexers) StripSecrets() UserDataIndexers {
	ud.Indexers = slices.Clone(ud.Indexers)
	for i := range ud.Indexers {
		s := &ud.Indexers[i]
		s.APIKey = ""
	}
	return ud
}

var jackettCache = cache.NewLRUCache[*jackett.Client](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "stremio:userdata:indexers:jackett",
})

var genericTorznabCache = cache.NewLRUCache[*generic.TorznabClient](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "stremio:userdata:indexers:generic",
})

func (ud *UserDataIndexers) Compress() {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		indexer.Compress()
	}
}

func (ud *UserDataIndexers) Decompress() {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		indexer.Decompress()
	}
}

func (ud *UserDataIndexers) Prepare() ([]torznab_client.Indexer, error) {
	indexers := make([]torznab_client.Indexer, 0, len(ud.Indexers))
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]

		baseURL := indexer.URL
		apiKey := indexer.APIKey

		switch indexer.Name {
		case IndexerNameGeneric:
			key := baseURL + ":" + apiKey
			var client *generic.TorznabClient
			if !genericTorznabCache.Get(key, &client) {
				client = generic.NewClient(&generic.TorznabClientConfig{
					BaseURL: baseURL,
					APIKey:  apiKey,
				})
				err := genericTorznabCache.Add(key, client)
				if err != nil {
					return indexers, err
				}
			}
			indexers = append(indexers, client)

		case IndexerNameJackett:
			u := jackett.TorznabURL(baseURL)
			if err := u.Parse(); err != nil {
				return indexers, err
			}

			key := u.BaseURL + ":" + apiKey
			var client *jackett.Client
			if !jackettCache.Get(key, &client) {
				client = jackett.NewClient(&jackett.ClientConfig{
					BaseURL: u.BaseURL,
					APIKey:  apiKey,
				})
				err := jackettCache.Add(key, client)
				if err != nil {
					return indexers, err
				}
			}
			c := client.GetTorznabClient(u.IndexerId)
			indexers = append(indexers, c)

		default:
			return indexers, errors.New("unsupported indexer: " + string(indexer.Name))
		}
	}
	return indexers, nil
}
