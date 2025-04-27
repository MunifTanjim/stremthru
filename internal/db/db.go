package db

import (
	"database/sql"
	"log"
	"net/url"

	"github.com/MunifTanjim/stremthru/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	URI ConnectionURI
}

var db *DB
var Dialect DBDialect

var BooleanFalse string
var CurrentTimestamp string
var FnJSONGroupArray string
var FnJSONObject string

var connUri, dsnModifiers = func() (ConnectionURI, []DSNModifier) {
	uri, err := ParseConnectionURI(config.DatabaseURI)
	if err != nil {
		log.Fatalf("[db] failed to parse uri: %v\n", err)
	}

	Dialect = uri.Dialect
	dsnModifiers := []DSNModifier{}

	switch Dialect {
	case DBDialectSQLite:
		BooleanFalse = "0"
		CurrentTimestamp = "unixepoch()"
		FnJSONGroupArray = "json_group_array"
		FnJSONObject = "json_object"

		dsnModifiers = append(dsnModifiers, func(u *url.URL, q *url.Values) {
			u.Scheme = "file"
		})
	case DBDialectPostgres:
		BooleanFalse = "false"
		CurrentTimestamp = "current_timestamp"
		FnJSONGroupArray = "json_agg"
		FnJSONObject = "json_build_object"
	default:
		log.Fatalf("[db] unsupported dialect: %v\n", Dialect)
	}

	return uri, dsnModifiers
}()

func Exec(query string, args ...any) (sql.Result, error) {
	return db.Exec(adaptQuery(query), args...)
}

func Query(query string, args ...any) (*sql.Rows, error) {
	return db.Query(adaptQuery(query), args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRow(adaptQuery(query), args...)
}

type Tx struct {
	tx *sql.Tx
}

func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.tx.Exec(adaptQuery(query), args...)
}

func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

func Begin() (*Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

func Ping() {
	err := db.Ping()
	if err != nil {
		log.Fatalf("[db] failed to ping: %v\n", err)
	}
	one := 0
	row := QueryRow("SELECT 1")
	if err := row.Scan(&one); err != nil {
		log.Fatalf("[db] failed to query: %v\n", err)
	}
}

func Open() *DB {
	database, err := sql.Open(connUri.driverName, connUri.DSN(dsnModifiers...))
	if err != nil {
		log.Fatalf("[db] failed to open: %v\n", err)
	}
	db = &DB{
		DB:  database,
		URI: connUri,
	}

	return db
}

func Close() error {
	return db.Close()
}
