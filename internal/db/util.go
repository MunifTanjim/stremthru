package db

import (
	"net/url"
	"strings"
)

type ConnectionURI struct {
	dialect          string
	driverName       string
	connectionString string
}

func ParseConnectionURI(connection_uri string) (ConnectionURI, error) {
	uri := ConnectionURI{}

	u, err := url.Parse(connection_uri)
	if err != nil {
		return uri, err
	}

	switch u.Scheme {
	case "sqlite":
		uri.dialect = "sqlite"
		uri.driverName = "libsql"
		u.Scheme = "file"
		uri.connectionString = strings.Replace(u.String(), "://", ":", 1)
	case "postgresql":
		uri.dialect = "postgres"
		uri.driverName = "pgx"
		uri.connectionString = u.String()
	}

	return uri, nil

}
