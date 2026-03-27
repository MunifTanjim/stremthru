package generic

import (
	"net/url"

	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var (
	_ torznab_client.Indexer = (*TorznabClient)(nil)
)

type TorznabClientConfig struct {
	BaseURL   string
	APIKey    string
	UserAgent string
	ID        int64
	Name      string
}

type TorznabClient struct {
	*torznab_client.Client
	id   string
	name string
}

func (tc TorznabClient) GetId() string {
	return tc.id
}

func (tc TorznabClient) GetName() string {
	return tc.name
}

func (tc TorznabClient) Search(query url.Values) ([]torznab_client.Torz, error) {
	caps, err := tc.GetCaps()
	if err != nil {
		return nil, err
	}
	params := &torznab_client.Ctx{}
	params.Query = &query
	var resp torznab_client.Response[SearchResponse]
	_, err = tc.Client.Request("GET", "/api", params, &resp)
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
		t := *item.ToTorz()
		if t.Indexer == "" {
			t.Indexer = caps.Server.Title
		}
		if t.Indexer == "" {
			t.Indexer = tc.GetName()
		}
		result = append(result, t)
	}
	return result, nil
}

func NewClient(conf *TorznabClientConfig) *TorznabClient {
	tc := torznab_client.NewClient(&torznab_client.ClientConfig{
		BaseURL:   conf.BaseURL,
		APIKey:    conf.APIKey,
		UserAgent: conf.UserAgent,
	})
	var id string
	if conf.ID != 0 {
		id = "generic/" + util.IntToString(conf.ID)
	} else {
		id = tc.BaseURL.Host
	}
	return &TorznabClient{Client: tc, id: id, name: conf.Name}
}
