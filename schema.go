package main

import (
	"embed"
	"log"
	"os"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/**/*.sql
var migrationsFS embed.FS

func RunSchemaMigration(uri db.ConnectionURI, database *db.DB) {
	l := log.New(os.Stderr, "=", 0)

	goose.SetBaseFS(migrationsFS)
	goose.SetTableName("db_migration_version")
	goose.SetLogger(log.New(os.Stderr, "=   ", 0))

	dir := ""
	switch uri.Dialect {
	case db.DBDialectSQLite:
		goose.SetDialect("sqlite")
		dir = "migrations/sqlite"
	case db.DBDialectPostgres:
		goose.SetDialect("postgres")
		dir = "migrations/postgres"
	}

	l.Println("====================")
	l.Println("= Database Schema ==")
	l.Println("====================")

	l.Println()
	l.Println(" STATE:")
	l.Println()
	if err := goose.Status(database.DB, dir); err != nil {
		l.Fatalf(" Failed to check state: %v\n", err)
	}

	l.Println()
	l.Println(" MIGRATION:")
	l.Println()
	if err := goose.Up(database.DB, dir); err != nil {
		l.Fatalf(" Failed to run migrations: %v\n", err)
	}

	l.Println()
	l.Print("====================\n\n")
}
