package tvdb

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"golang.org/x/oauth2"
)

var apiClientCache = cache.NewLRUCache[APIClient](&cache.CacheConfig{
	Lifetime: 1 * time.Hour,
	Name:     "tvdb:api-client",
})

func GetAPIClient() *APIClient {
	tokenId := config.Integration.TVDB.SystemOAuthTokenId

	var cachedClient APIClient
	if apiClientCache.Get(tokenId, &cachedClient) {
		return &cachedClient
	}

	conf := APIClientConfig{}

	conf.OAuth = &APIClientConfigOAuth{
		GetTokenSource: func(oauthConfig oauth2.Config) oauth2.TokenSource {
			otok, _ := oauth.GetOAuthTokenById(tokenId)
			if otok == nil {
				if tokenId == config.Integration.TVDB.SystemOAuthTokenId {
					tok, err := oauth.TVDBOAuthConfig.PasswordCredentialsToken("", config.Integration.TVDB.APIKey)
					if err != nil {
						panic(err)
					}
					otok = &oauth.OAuthToken{}
					otok = otok.FromToken(tok)
				}
			}
			return oauth.DatabaseTokenSource(&oauth.DatabaseTokenSourceConfig{
				OAuth:             &oauthConfig,
				TokenSourceConfig: oauth.TVDBTokenSourceConfig,
			}, otok.ToToken())
		},
	}

	client := NewAPIClient(&conf)

	apiClientCache.Add(tokenId, *client)

	return client
}
