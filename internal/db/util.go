package db

import (
	"net/url"
	"strings"
)

type ConnectionURIVariant string

const (
	ConnectionURIVariantSQLite     ConnectionURIVariant = "sqlite"
	ConnectionURIVariantPostgreSQL ConnectionURIVariant = "postgresql"
)

type ConnectionURI struct {
	variant ConnectionURIVariant
	value   string
}

func ParseConnectionURI(connection_uri string) (ConnectionURI, error) {
	uri := ConnectionURI{}

	u, err := url.Parse(connection_uri)
	if err != nil {
		return uri, err
	}

	switch u.Scheme {
	case "sqlite":
		uri.variant = ConnectionURIVariantSQLite
		u.Scheme = "file"
		uri.value = strings.Replace(u.String(), "://", ":", 1)
	case "postgresql":
		uri.variant = ConnectionURIVariantPostgreSQL
		uri.value = u.String()
	}

	return uri, nil

}
