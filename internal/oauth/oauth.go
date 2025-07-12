package oauth

import "golang.org/x/oauth2"

type OAuthConfig struct {
	oauth2.Config
	AuthCodeURL              func(state string, opts ...oauth2.AuthCodeOption) string
	Exchange                 func(code, state string) (*oauth2.Token, error)
	PasswordCredentialsToken func(username, password string) (*oauth2.Token, error)
	TryAuthCodeURL           func(state string, opts ...oauth2.AuthCodeOption) (string, error)
}
