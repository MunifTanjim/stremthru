package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

// dirForDatabaseURI returns the directory that should hold imdb.db: same
// directory as the main sqlite database file when the main DB is sqlite,
// else dataDir.
func dirForDatabaseURI(databaseURI, dataDir string) string {
	uri, err := db.ParseConnectionURI(databaseURI)
	if err != nil || uri.Dialect != db.DBDialectSQLite {
		return dataDir
	}
	// sqlite://./data/x.db parses as host "." + path "/data/x.db", i.e. a
	// path relative to the working directory — not an absolute /data path.
	p := uri.Path
	if uri.Host == "." {
		p = "." + p
	}
	if !filepath.IsAbs(p) {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
	}
	return filepath.Dir(p)
}

// ResolvePath returns the imdb.db file path: same directory as the main
// sqlite database file when the main DB is sqlite, else config.DataDir.
func ResolvePath() string {
	return filepath.Join(dirForDatabaseURI(config.DatabaseURI, config.DataDir), "imdb.db")
}

type DB struct {
	*sql.DB
	path string
}

func dsnForPath(path string) string {
	q := url.Values{}
	q.Set("mode", "rwc")
	q.Set("_busy_timeout", "5000")
	q.Set("_cache_size", "-64000")
	q.Set("_journal_mode", "WAL")
	q.Set("_loc", "UTC")
	q.Set("_synchronous", "NORMAL")
	q.Set("_txlock", "immediate")
	return "file:" + path + "?" + q.Encode()
}

func Open(path string) (*DB, error) {
	if path == "" {
		path = ResolvePath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("imdb_meta/db: failed to create directory: %w", err)
	}
	database, err := sql.Open("sqlite3", dsnForPath(path))
	if err != nil {
		return nil, fmt.Errorf("imdb_meta/db: failed to open: %w", err)
	}
	database.SetMaxOpenConns(8)
	if _, err := database.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=OFF;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("imdb_meta/db: failed to set pragmas: %w", err)
	}
	return &DB{DB: database, path: path}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) Ping() error {
	if err := db.DB.Ping(); err != nil {
		return err
	}
	var one int
	return db.QueryRow(`SELECT 1`).Scan(&one)
}
