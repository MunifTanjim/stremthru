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
	database, err := sql.Open("libsql", config.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db %s", err)
	}
	db = database
	return db
}

func Close() {
	db.Close()
}
