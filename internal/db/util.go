package db

import (
	"errors"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type DBDialect string

const (
	DBDialectPostgres DBDialect = "postgres"
	DBDialectSQLite   DBDialect = "sqlite"
)

type ConnectionURI struct {
	*url.URL
	driverName string
	Dialect    DBDialect
}

type DSNModifier func(u *url.URL, q *url.Values)

func (uri ConnectionURI) DSN(mods ...DSNModifier) string {
	u, err := url.Parse(uri.URL.String())
	if err != nil {
		log.Fatalf("failed to generate dsn: %v\n", err)
	}

	switch u.Scheme {
	case "sqlite":
		q := u.Query()
		q.Add("mode", "rwc")
		q.Add("_busy_timeout", "5000")
		q.Add("_journal_mode", "WAL")
		q.Add("_txlock", "immediate")
		q.Add("_loc", "UTC")
		for _, mod := range mods {
			mod(u, &q)
		}
		u.RawQuery = q.Encode()
		dsn := u.String()
		if u.Scheme == "file" {
			dsn = strings.Replace(dsn, "file://", "file:", 1)
		}
		return dsn
	case "postgresql":
		return u.String()
	default:
		return u.String()
	}
}

func ParseConnectionURI(connection_uri string) (ConnectionURI, error) {
	uri := ConnectionURI{}

	u, err := url.Parse(connection_uri)
	if err != nil {
		return uri, err
	}

	uri.URL = u

	switch u.Scheme {
	case "sqlite":
		uri.Dialect = DBDialectSQLite
		uri.driverName = "sqlite3"
		if u.Host != "" && u.Host != "." {
			return uri, errors.New("invalid path, must start with '/' or './'")
		}
	case "postgresql":
		uri.Dialect = DBDialectPostgres
		uri.driverName = "pgx"
	default:
		return uri, errors.New("unsupported scheme: " + u.Scheme)
	}

	return uri, nil
}

func adaptQuery(query string) string {
	if Dialect == DBDialectSQLite {
		return query
	}

	var q strings.Builder
	pos := 1

	for _, char := range query {
		if char == '?' {
			q.WriteRune('$')
			q.WriteString(strconv.Itoa(pos))
			pos++
		} else {
			q.WriteRune(char)
		}
	}

	return q.String()
}

func JoinColumnNames(columns ...string) string {
	return `"` + strings.Join(columns, `","`) + `"`
}

func JoinPrefixedColumnNames(prefix string, columns ...string) string {
	return prefix + `"` + strings.Join(columns, `",`+prefix+`"`) + `"`
}

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-z0-9]`)
var whitespacesRegex = regexp.MustCompile(`\s{2,}`)
var fts5SymbolRegex = regexp.MustCompile(`[-+*:^]`)

func PrepareFTS5Query(query string, lenient bool) string {
	query = whitespacesRegex.ReplaceAllLiteralString(fts5SymbolRegex.ReplaceAllLiteralString(strings.ReplaceAll(query, `"`, `""`), " "), " ")
	if strings.TrimSpace(query) == "" {
		return ""
	}
	sep := `" "`
	if lenient {
		sep = `" OR "`
	}
	return `"` + strings.Join(strings.Split(query, " "), sep) + `"`
}
