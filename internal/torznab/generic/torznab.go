package generic

import (
	"net/url"

	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
)

type TorznabClientConfig struct {
	BaseURL   string
	APIKey    string
	UserAgent string
}

type TorznabClient struct {
	*torznab_client.Client
	id string
}

func (tc TorznabClient) GetId() string {
	return "generic/" + tc.id
}

func (tc TorznabClient) Search(query url.Values) ([]torznab_client.Torz, error) {
	params := &torznab_client.Ctx{}
	params.Query = &query
	var resp torznab_client.Response[SearchResponse]
	_, err := tc.Client.Request("GET", "", params, &resp)
	if err != nil {
		return nil, err
	}
	items := resp.Data.Channel.Items
	result := make([]torznab_client.Torz, 0, len(items))
	for i := range items {
		item := &items[i]
		if item.Enclosure.Length == 0 {
			continue
		}
		result = append(result, *item.ToTorz())
	}
	return result, nil
}

func NewClient(conf *TorznabClientConfig) *TorznabClient {
	tc := torznab_client.NewClient(&torznab_client.ClientConfig{
		BaseURL:   conf.BaseURL,
		APIKey:    conf.APIKey,
		UserAgent: conf.UserAgent,
	})
	u := tc.BaseURL
	id := u.Host + u.Path
	return &TorznabClient{Client: tc, id: id}
}
