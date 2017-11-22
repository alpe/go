// +build integration

package database

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
