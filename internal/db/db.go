package db

import (
	"database/sql"
	"log"

	"github.com/MunifTanjim/stremthru/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var dialect string

var CurrentTimestamp string

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

func Open() *sql.DB {
	uri, err := ParseConnectionURI(config.DatabaseURI)
	if err != nil {
		log.Fatalf("[db] failed to parse uri: %v\n", err)
	}

	dialect = uri.dialect
	switch dialect {
	case "sqlite":
		CurrentTimestamp = "unixepoch()"
	case "postgres":
		CurrentTimestamp = "current_timestamp"
	default:
		log.Fatalf("[db] unsupported dialect: %v\n", dialect)
	}

	database, err := sql.Open(uri.driverName, uri.connectionString)
	if err != nil {
		log.Fatalf("[db] failed to open: %v\n", err)
	}
	db = database

	return db
}

func Close() error {
	return db.Close()
}
