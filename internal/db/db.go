package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/url"
	"sync/atomic"
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

	replicas       []*sql.DB
	replicaIdx     uint32
	replicaOnClose []func() error
}

func (db *DB) getReplica() *sql.DB {
	n := len(db.replicas)
	if n == 0 {
		return db.DB
	}
	i := atomic.AddUint32(&db.replicaIdx, 1)
	return db.replicas[int(i)%n]
}

var db = &DB{}
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
	return db.getReplica().Query(adaptQuery(query), args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.getReplica().QueryRow(adaptQuery(query), args...)
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
	for i, replica := range db.replicas {
		if err := replica.Ping(); err != nil {
			log.Fatalf("[db] failed to ping replica[%d]: %v\n", i, err)
		}
		row := replica.QueryRow(adaptQuery("SELECT 1"))
		if err := row.Scan(&one); err != nil {
			log.Fatalf("[db] failed to query replica[%d]: %v\n", i, err)
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
			err := db.DB.Close()
			pool.Close()
			return err
		}
	}

	db.URI = connUri

	for _, replicaUri := range config.DatabaseReplicaURIs {
		if connUri.Dialect != DBDialectPostgres {
			log.Fatalf("[db] replica not supported for %s\n", connUri.Dialect)
		}

		replicaUri, err := ParseConnectionURI(replicaUri)
		if err != nil {
			log.Fatalf("[db] failed to parse replica uri: %v\n", err)
		}
		if replicaUri.Dialect != connUri.Dialect {
			log.Fatalf("[db] replica dialect mismatch: %s\n", replicaUri.Dialect)
		}
		pool, err := pgxpool.New(context.Background(), replicaUri.DSN())
		if err != nil {
			log.Fatalf("[db] failed to create replica connection pool: %v\n", err)
		}
		replica := stdlib.OpenDBFromPool(pool)
		db.replicas = append(db.replicas, replica)
		db.replicaOnClose = append(db.replicaOnClose, func() error {
			err := replica.Close()
			pool.Close()
			return err
		})
	}

	return db
}

func Close() error {
	errs := []error{}
	if db.onClose != nil {
		errs = append(errs, db.onClose())
	}
	for _, onClose := range db.replicaOnClose {
		if onClose != nil {
			errs = append(errs, onClose())
		}
	}
	return errors.Join(errs...)
}
