package db

import "database/sql"

// SetForTesting redirects db queries to sqlDB, undone via cleanup (pass t.Cleanup).
func SetForTesting(sqlDB *sql.DB, cleanup func(func())) {
	prev := db.DB
	db.DB = sqlDB
	cleanup(func() {
		db.DB = prev
		sqlDB.Close()
	})
}
