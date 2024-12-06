package db

import (
	"database/sql"
	"log"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/tursodatabase/go-libsql"
)

var db *sql.DB
var dialect string

var CurrentTimestamp string

func adaptQuery(query string) string {
	if dialect == "sqlite" {
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

func Exec(query string, args ...any) (sql.Result, error) {
	return db.Exec(adaptQuery(query), args...)
}

func Query(query string, args ...any) (*sql.Rows, error) {
	return db.Query(adaptQuery(query), args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRow(adaptQuery(query), args...)
}

func Ping() {
	err := db.Ping()
	if err != nil {
		log.Fatalf("[db] failed to ping: %v\n", err)
	}
	_, err = Query("SELECT 1")
	if err != nil {
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

	if dialect == "sqlite" {
		result := db.QueryRow("PRAGMA journal_mode=WAL")
		if err := result.Err(); err != nil {
			log.Fatalf("[db] failed to enable WAL mode: %v\n", err)
		}
		journal_mode := ""
		err := result.Scan(&journal_mode)
		if err != nil {
			log.Fatalf("[db] failed to enable WAL mode: %v\n", err)
		}
		log.Printf("[db] journal_mode: %v\n", journal_mode)
	}

	return db
}

func Close() {
	db.Close()
}
