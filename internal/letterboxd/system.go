package letterboxd

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"golang.org/x/oauth2"
)

var apiClientCache = cache.NewLRUCache[APIClient](&cache.CacheConfig{
	Lifetime: 1 * time.Hour,
	Name:     "letterboxd:api-client",
})

func GetSystemClient() *APIClient {
	cacheKey := "system"
	var cachedClient APIClient
	if apiClientCache.Get(cacheKey, &cachedClient) {
		return &cachedClient
	}

	client := NewAPIClient(&APIClientConfig{
		OAuth: &APIClientConfigOAuth{
			GetTokenSource: func(oauthConfig oauth2.Config) oauth2.TokenSource {
				tok, err := oauth.LetterboxdOAuthConfig.ClientCredentialsToken(config.Integration.Letterboxd.ClientId, config.Integration.Letterboxd.ClientSecret)
				if err != nil {
					log.Error("failed to get letterboxd client credentials token", "error", err)
					return nil
				}
				return oauth.DatabaseTokenSource(&oauth.DatabaseTokenSourceConfig{
					OAuth:             &oauthConfig,
					TokenSourceConfig: oauth.LetterboxdTokenSourceConfig,
					GetUser: func(tok *oauth2.Token, tsc *oauth.TokenSourceConfig, oauthConfig *oauth2.Config) (userId string, userName string, err error) {
						return config.Integration.Letterboxd.ClientId, "", nil
					},
					Refresh: func(ts oauth2.TokenSource) (*oauth2.Token, error) {
						return oauth.LetterboxdOAuthConfig.ClientCredentialsToken(config.Integration.Letterboxd.ClientId, config.Integration.Letterboxd.ClientSecret)
					},
				}, tok)
			},
		},
	})

	apiClientCache.Add(cacheKey, *client)

	return client
}
