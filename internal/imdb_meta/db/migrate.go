package db

import (
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate sets goose's package-level FS, table name and dialect, which are
// shared with the main database migrations. It must not run concurrently with
// them.
func (db *DB) Migrate() error {
	goose.SetBaseFS(migrationsFS)
	goose.SetTableName("imdb_migration_version")
	if err := goose.SetDialect("sqlite"); err != nil {
		return err
	}
	return goose.Up(db.DB, "migrations")
}
