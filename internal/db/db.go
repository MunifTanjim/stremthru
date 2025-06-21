package db

import (
	"database/sql"
	"log"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	URI ConnectionURI
}

var db = &DB{}
var Dialect DBDialect

var BooleanFalse string
var BooleanTrue string
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
		BooleanTrue = "1"
		CurrentTimestamp = "unixepoch()"
		FnJSONGroupArray = "json_group_array"
		FnJSONObject = "json_object"

		dsnModifiers = append(dsnModifiers, func(u *url.URL, q *url.Values) {
			u.Scheme = "file"
		})
	case DBDialectPostgres:
		BooleanFalse = "false"
		BooleanTrue = "true"
		CurrentTimestamp = "current_timestamp"
		FnJSONGroupArray = "json_agg"
		FnJSONObject = "json_build_object"
	default:
		log.Fatalf("[db] unsupported dialect: %v\n", Dialect)
	}

	return uri, dsnModifiers
}()

type dbExec func(query string, args ...any) (sql.Result, error)
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

var getExec = func(db Executor) dbExec {
	if Dialect == DBDialectPostgres {
		return func(query string, args ...any) (sql.Result, error) {
			return db.Exec(adaptQuery(query), args...)
		}
	}

	return func(query string, args ...any) (sql.Result, error) {
		retryLeft := 2
		r, err := db.Exec(query, args...)
		for err != nil && retryLeft > 0 {
			if e, ok := err.(sqlite3.Error); ok && e.Code == sqlite3.ErrBusy {
				time.Sleep(2 * time.Second)
				r, err = db.Exec(query, args...)
				retryLeft--
			} else {
				retryLeft = 0
			}
		}
		return r, err
	}
}

var Exec = getExec(db)

func Query(query string, args ...any) (*sql.Rows, error) {
	return db.Query(adaptQuery(query), args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRow(adaptQuery(query), args...)
}

type Tx struct {
	tx   *sql.Tx
	exec dbExec
}

func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.exec(query, args...)
}

func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

func Begin() (*Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, exec: getExec(tx)}, nil
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
	*db = *&DB{
		DB:  database,
		URI: connUri,
	}

	return db
}

func Close() error {
	return db.Close()
}
