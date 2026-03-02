package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	URI     ConnectionURI
	onClose func() error
}

var db = &DB{}
var readDB *sql.DB   // read replica, nil when not configured
var readDBClose func() // cleanup function for read replica
var Dialect DBDialect

var BooleanFalse string
var BooleanTrue string
var CurrentTimestamp string
var FnJSONGroupArray string
var FnJSONObject string
var FnMax string
var NewAdvisoryLock func(names ...string) AdvisoryLock

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
		FnMax = "max"
		NewAdvisoryLock = sqliteNewAdvisoryLock

		dsnModifiers = append(dsnModifiers, func(u *url.URL, q *url.Values) {
			u.Scheme = "file"
		})
	case DBDialectPostgres:
		BooleanFalse = "false"
		BooleanTrue = "true"
		CurrentTimestamp = "current_timestamp"
		FnJSONGroupArray = "json_agg"
		FnJSONObject = "json_build_object"
		FnMax = "greatest"
		NewAdvisoryLock = postgresNewAdvisoryLock
	default:
		log.Fatalf("[db] unsupported dialect: %v\n", Dialect)
	}

	return uri, dsnModifiers
}()

type dbExec func(query string, args ...any) (sql.Result, error)
type dbQuery func(query string, args ...any) (*sql.Rows, error)
type dbQueryRow func(query string, args ...any) *sql.Row
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
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
	q := adaptQuery(query)
	if readDB != nil {
		return readDB.Query(q, args...)
	}
	return db.Query(q, args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	q := adaptQuery(query)
	if readDB != nil {
		return readDB.QueryRow(q, args...)
	}
	return db.QueryRow(q, args...)
}

type dbExecutor struct{}

func (e dbExecutor) Exec(query string, args ...any) (sql.Result, error) {
	return Exec(query, args...)
}

func (e dbExecutor) Query(query string, args ...any) (*sql.Rows, error) {
	return Query(query, args...)
}

func (e dbExecutor) QueryRow(query string, args ...any) *sql.Row {
	return QueryRow(query, args...)
}

var execturor = dbExecutor{}

func GetDB() *dbExecutor {
	return &execturor
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

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.Query(adaptQuery(query), args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.tx.QueryRow(adaptQuery(query), args...)
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
	row := db.QueryRow(adaptQuery("SELECT 1"))
	if err := row.Scan(&one); err != nil {
		log.Fatalf("[db] failed to query: %v\n", err)
	}
	if readDB != nil {
		if err := readDB.Ping(); err != nil {
			log.Fatalf("[db] failed to ping read replica: %v\n", err)
		}
		row := readDB.QueryRow(adaptQuery("SELECT 1"))
		if err := row.Scan(&one); err != nil {
			log.Fatalf("[db] failed to query read replica: %v\n", err)
		}
	}
}

func Open() *DB {
	switch connUri.Dialect {
	case DBDialectSQLite:
		database, err := sql.Open(connUri.DriverName, connUri.DSN(dsnModifiers...))
		if err != nil {
			log.Fatalf("[db] failed to open: %v\n", err)
		}
		db.DB = database
	case DBDialectPostgres:
		pool, err := pgxpool.New(context.Background(), connUri.DSN())
		if err != nil {
			log.Fatalf("[db] failed to create connection pool: %v\n", err)
		}
		db.DB = stdlib.OpenDBFromPool(pool)
		db.onClose = func() error {
			pool.Close()
			return nil
		}
	}

	db.URI = connUri

	if config.DatabaseReadReplicaURI != "" {
		replicaUri, err := ParseConnectionURI(config.DatabaseReadReplicaURI)
		if err != nil {
			log.Fatalf("[db] failed to parse read replica uri: %v\n", err)
		}
		if replicaUri.Dialect != DBDialectPostgres {
			log.Fatalf("[db] read replica only supports postgresql\n")
		}
		pool, err := pgxpool.New(context.Background(), replicaUri.DSN())
		if err != nil {
			log.Fatalf("[db] failed to create read replica connection pool: %v\n", err)
		}
		readDB = stdlib.OpenDBFromPool(pool)
		readDBClose = func() {
			readDB.Close()
			pool.Close()
		}
		log.Println("[db] read replica enabled")
	}

	return db
}

func Close() error {
	if readDBClose != nil {
		readDBClose()
	}
	err := db.Close()
	if db.onClose != nil {
		err = errors.Join(err, db.onClose())
	}
	return err
}
