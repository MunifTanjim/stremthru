package util

import (
	"errors"
	"strings"
)

type BasicAuth struct {
	Username string
	Password string
	Token    string
}

func ParseBasicAuth(token string) (BasicAuth, error) {
	basicAuth := BasicAuth{}
	token = strings.TrimSpace(token)
	if strings.ContainsRune(token, ':') {
		username, password, _ := strings.Cut(token, ":")
		basicAuth.Username = username
		basicAuth.Password = password
		basicAuth.Token = Base64Encode(token)
	} else if decoded, err := Base64Decode(token); err == nil {
		if username, password, ok := strings.Cut(strings.TrimSpace(decoded), ":"); ok {
			basicAuth.Username = username
			basicAuth.Password = password
			basicAuth.Token = token
		} else {
			return basicAuth, errors.New("invalid token")
		}
	} else {
		return basicAuth, errors.New("malformed token")
	}
	return basicAuth, nil
}
