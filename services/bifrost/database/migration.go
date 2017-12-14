package database

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
)

// RunMigrations executes all database migration files in given source path.
func RunMigrations(db *sql.DB, dialect, sourcePath string) error {
	migrations := &migrate.FileMigrationSource{
		Dir: sourcePath,
	}
	_, err := migrate.Exec(db, dialect, migrations, migrate.Up)
	return err
}
