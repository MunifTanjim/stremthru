package jackett

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
)

type torznabURL struct {
	raw       string
	BaseURL   string
	IndexerId string
}

func TorznabURL(str string) *torznabURL {
	purl := torznabURL{raw: str}
	return &purl
}

func (turl *torznabURL) Parse() error {
	if turl.BaseURL != "" && turl.IndexerId != "" {
		return nil
	}
	if strings.HasPrefix(turl.raw, "http://") || strings.HasPrefix(turl.raw, "https://") {
		return turl.FromDecoded()
	}
	return turl.FromEncoded()
}

func (purl *torznabURL) FromDecoded() error {
	for _, match := range torznabUrlPattern.FindAllStringSubmatch(purl.raw, -1) {
		for i, name := range torznabUrlPattern.SubexpNames() {
			value := match[i]
			switch name {
			case "base_url":
				purl.BaseURL = value
			case "indexer_id":
				purl.IndexerId = value
			}
		}
	}
	if purl.BaseURL == "" || purl.IndexerId == "" {
		return errors.New("invalid torznab url")
	}
	return nil
}

func (purl *torznabURL) FromEncoded() error {
	if strings.Contains(purl.raw, ":::") {
		indexerId, baseUrl, ok := strings.Cut(purl.raw, ":::")
		if !ok {
			return errors.New("invalid encoded torznab url")
		}
		purl.BaseURL = baseUrl
		purl.IndexerId = indexerId
		return nil
	}

	schemeHost, indexerId, ok := strings.Cut(purl.raw, "::")
	if !ok {
		return errors.New("invalid encoded torznab url")
	}
	scheme, host, ok := strings.Cut(schemeHost, ":")
	if !ok {
		return errors.New("invalid encoded torznab url")
	}
	purl.BaseURL = scheme + "://" + host
	purl.IndexerId = indexerId
	return nil
}

func (purl torznabURL) Encode() string {
	if err := purl.Parse(); err != nil {
		return ""
	}
	u, err := url.Parse(purl.BaseURL)
	if err != nil {
		return ""
	}
	return u.Scheme + ":" + u.Host + "::" + purl.IndexerId
}

func (purl torznabURL) Decode() string {
	if strings.HasPrefix(purl.raw, "http://") || strings.HasPrefix(purl.raw, "https://") {
		return purl.raw
	}
	if err := purl.Parse(); err != nil {
		return ""
	}
	return strings.TrimRight(purl.BaseURL, "/") + "/api/v2.0/indexers/" + purl.IndexerId + "/results/torznab"
}

var torznabUrlPattern = regexp.MustCompile(`(?i)^(?<base_url>https?:\/\/.+?)\/api\/v2\.0\/indexers/(?<indexer_id>[^\/]+)\/results\/torznab\/?$`)

func ParseTorznabURL(torznabUrl string) (*torznabURL, error) {
	r := torznabURL{raw: torznabUrl}
	err := r.FromDecoded()
	return &r, err
}

type TorznabClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	UserAgent  string
	IndexerId  string
}

type TorznabClient struct {
	*torznab_client.Client
	id string
}

func (tc TorznabClient) GetId() string {
	return "jackett/" + tc.id
}

func (tc TorznabClient) Search(query *torznab_client.Query) ([]torznab_client.Torz, error) {
	params := &Ctx{}
	q := query.Values()
	params.Query = &q
	var resp torznab_client.Response[SearchResponse]
	_, err := tc.Client.Request("GET", "/api", params, &resp)
	if err != nil {
		return nil, err
	}
	items := resp.Data.Channel.Items
	result := make([]torznab_client.Torz, 0, len(items))
	for i := range items {
		item := &items[i]
		if item.Size == 0 && item.Grabs == 0 && item.Enclosure.Length == 0 {
			continue
		}
		result = append(result, *item.ToTorz())
	}
	return result, nil
}

func (c *Client) GetTorznabClient(id string) *TorznabClient {
	if id == "" {
		id = "all"
	}
	var client TorznabClient
	if c.torznabClientById.Get(id, &client) {
		return &client
	}
	tc := torznab_client.NewClient(&torznab_client.ClientConfig{
		BaseURL:    c.BaseURL.JoinPath("/api/v2.0/indexers/" + id + "/results/torznab").String(),
		HTTPClient: c.HTTPClient,
		APIKey:     c.apiKey,
		UserAgent:  c.userAgent,
	})
	client = TorznabClient{Client: tc, id: id}
	c.torznabClientById.Add(id, client)
	return &client
}
