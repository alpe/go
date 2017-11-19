// +build integration

package database

import (
	"database/sql"
	"flag"
	"sync"
	"testing"

	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"
)

var dsn = flag.String("db-url", "postgres://root:mysecretpassword@127.0.0.1:5432/circle_test?sslmode=disable", "database url")
var once sync.Once

func TestSetupDB(t *testing.T) {
	OpenTestDB(t)
}
func OpenTestDB(t *testing.T) *sql.DB {
	flag.Parse()
	db, err := sql.Open("postgres", *dsn)
	require.NoError(t, err)
	once.Do(migrateSchema(t, db))
	return db
}

func migrateSchema(t *testing.T, db *sql.DB) func() {
	return func() {
		migrations := &migrate.FileMigrationSource{
			Dir: "./migrations",
		}
		_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
		require.NoError(t, err)
	}
}
