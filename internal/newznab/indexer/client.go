package newznab_indexer

import (
	"fmt"

	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
)

func (i *NewznabIndexer) GetClient() (*newznab_client.Client, error) {
	apiKey, err := i.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt api key: %w", err)
	}

	client := newznab_client.NewClient(&newznab_client.ClientConfig{
		BaseURL: i.URL,
		APIKey:  apiKey,
	})

	return client, nil
}
