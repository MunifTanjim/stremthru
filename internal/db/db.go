package db

import (
	"database/sql"
	"log"

	"github.com/MunifTanjim/stremthru/internal/config"
	_ "github.com/tursodatabase/go-libsql"
)

var db *sql.DB

func Ping() {
	err := db.Ping()
	if err != nil {
		log.Fatalf("failed to ping db: %v\n", err)
	}
	_, err = db.Query("SELECT 1")
	if err != nil {
		log.Fatalf("failed to query db: %v\n", err)
	}
}

func Open() *sql.DB {
	uri, err := ParseConnectionURI(config.DatabaseURI)
	if err != nil {
		log.Fatalf("failed to open db %s", err)
	}

	database, err := sql.Open("libsql", uri.value)
	if err != nil {
		log.Fatalf("failed to open db %s", err)
	}
	db = database
	return db
}

func Close() {
	db.Close()
}
