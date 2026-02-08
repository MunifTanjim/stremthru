package qbittorrent

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

var DefaultHTTPClient = config.DefaultHTTPClient

type qbitConfig struct {
	URL         string // qBittorrent WebUI base URL
	Username    string
	Password    string
	FileBaseURL string // HTTP file server base URL for serving downloaded files
}

func parseToken(token string) (*qbitConfig, error) {
	parts := strings.SplitN(token, "|", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid qbittorrent token: expected 4 pipe-delimited parts (url|user|pass|file_base_url), got %d", len(parts))
	}
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("invalid qbittorrent token: part %d is empty", i)
		}
	}
	return &qbitConfig{
		URL:         strings.TrimRight(parts[0], "/"),
		Username:    parts[1],
		Password:    parts[2],
		FileBaseURL: strings.TrimRight(parts[3], "/"),
	}, nil
}

type sessionEntry struct {
	client  *http.Client
	expires time.Time
}

type APIClientConfig struct {
	HTTPClient *http.Client
}

type APIClient struct {
	HTTPClient *http.Client
	sessions   sync.Map // map[string]*sessionEntry
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	httpClient := conf.HTTPClient
	if httpClient == nil {
		httpClient = DefaultHTTPClient
	}
	return &APIClient{
		HTTPClient: httpClient,
	}
}

func (c *APIClient) sessionKey(cfg *qbitConfig) string {
	return cfg.URL + "|" + cfg.Username
}

func (c *APIClient) getOrCreateSession(cfg *qbitConfig) (*http.Client, error) {
	key := c.sessionKey(cfg)

	if entry, ok := c.sessions.Load(key); ok {
		se := entry.(*sessionEntry)
		if time.Now().Before(se.expires) {
			return se.client, nil
		}
	}

	return c.login(cfg)
}

func (c *APIClient) login(cfg *qbitConfig) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 90 * time.Second,
	}
	if c.HTTPClient != nil && c.HTTPClient.Transport != nil {
		client.Transport = c.HTTPClient.Transport
	}

	form := url.Values{
		"username": {cfg.Username},
		"password": {cfg.Password},
	}

	loginURL := cfg.URL + "/api/v2/auth/login"
	resp, err := client.Post(loginURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, UpstreamErrorWithCause(fmt.Errorf("qbittorrent login failed: %w", err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, UpstreamErrorWithCause(newQbitError(resp.StatusCode, body))
	}

	baseURL, _ := url.Parse(cfg.URL)
	cookies := jar.Cookies(baseURL)
	hasSID := false
	for _, cookie := range cookies {
		if cookie.Name == "SID" {
			hasSID = true
			break
		}
	}
	if !hasSID {
		return nil, UpstreamErrorWithCause(fmt.Errorf("qbittorrent login failed: no SID cookie received (body: %s)", string(body)))
	}

	key := c.sessionKey(cfg)
	se := &sessionEntry{
		client:  client,
		expires: time.Now().Add(55 * time.Minute),
	}
	c.sessions.Store(key, se)

	return client, nil
}

func (c *APIClient) invalidateSession(cfg *qbitConfig) {
	c.sessions.Delete(c.sessionKey(cfg))
}

func (c *APIClient) doRequest(cfg *qbitConfig, method, path string, form url.Values) (*http.Response, []byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		client, err := c.getOrCreateSession(cfg)
		if err != nil {
			return nil, nil, err
		}

		reqURL := cfg.URL + path
		var body io.Reader
		if form != nil && method == http.MethodPost {
			body = strings.NewReader(form.Encode())
		}

		req, err := http.NewRequest(method, reqURL, body)
		if err != nil {
			return nil, nil, err
		}
		if form != nil {
			if method == http.MethodGet {
				req.URL.RawQuery = form.Encode()
			} else {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return resp, nil, err
		}

		// Retry once on 403 (session expired)
		if resp.StatusCode == http.StatusForbidden && attempt == 0 {
			c.invalidateSession(cfg)
			continue
		}

		return resp, respBody, nil
	}
	return nil, nil, fmt.Errorf("qbittorrent request failed after retry")
}
