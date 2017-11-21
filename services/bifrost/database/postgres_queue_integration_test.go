// +build integration

package database

import (
	"database/sql"
	"flag"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dsn = flag.String("db-url", "postgres://root:mysecretpassword@127.0.0.1:5432/circle_test?sslmode=disable", "database url")
var once sync.Once

func TestIsEmptyShouldReturnFalseWhenTransactionsArePending(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	require.NoError(t, dbQueue.QueueAdd(queue.Transaction{
		TransactionID:    fmt.Sprintf("anyTx-%d", time.Now().UnixNano()),
		AssetCode:        "myAsset",
		Amount:           "100",
		StellarPublicKey: "myStellarPublicKeyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}))
	// when
	result, err := dbQueue.IsEmpty()
	// then
	require.NoError(t, err)
	assert.False(t, result)
}

// test is empty ignores all locked ones
// test is empty ignores all with failure count above threshold

func TestWithQueuedTransaction(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	require.NoError(t, dbQueue.QueueAdd(queue.Transaction{
		TransactionID:    fmt.Sprintf("anyTx-%d", time.Now().UnixNano()),
		AssetCode:        "myAsset",
		Amount:           "100",
		StellarPublicKey: "myStellarPublicKeyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}))

	// when
	var callbackExecuted bool
	myHandler := func(transaction queue.Transaction) error {
		callbackExecuted = true
		return nil
	}
	err := dbQueue.WithQueuedTransaction(myHandler)
	// then
	require.NoError(t, err)
	assert.True(t, callbackExecuted)
}

// test lock set
// test lock released

func OpenTestDB(t *testing.T) *sqlx.DB {
	flag.Parse()
	db, err := sqlx.Open("postgres", *dsn)
	require.NoError(t, err)
	once.Do(migrateSchema(t, db.DB))
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
