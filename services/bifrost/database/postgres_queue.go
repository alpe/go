package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

var _ queue.Queue = &PostgresDatabase{}

// QueueAdd implements queue.Queue interface. If element already exists in a queue, it should
// return nil.
func (d *PostgresDatabase) QueueAdd(tx queue.Transaction) error {
	transactionsQueueTable := d.getTable(transactionsQueueTableName, nil)
	transactionQueue := fromQueueTransaction(tx)
	_, err := transactionsQueueTable.Insert(transactionQueue).Exec()
	if err != nil {
		if isDuplicateError(err) {
			return nil
		}
	}
	return err
}

func (d *PostgresDatabase) IsEmpty() (bool, error) {
	var result bool
	err := withTx(d.session, readOnly(func(s *db.Session) error {
		rows, err := s.Query(squirrel.Select("count(id) = 0").
			From(transactionsQueueTableName).
			Where("pooled = false AND (locked_until is null OR locked_until < ?)"+
				" AND failure_count < ?", time.Now(), maxProcessingFailureCount))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			return rows.Scan(&result)
		}
		return nil
	}))
	return result, err
}

// QueuePool receives and removes the head of this queue. Returns nil if no elements found.
// QueuePool implements queue.Queue interface.
func (d *PostgresDatabase) WithQueuedTransaction(transactionHandler func(queue.Transaction) error) error {
	var row transactionsQueueRow
	var sessionToken string
	err := withTx(d.session, func(s *db.Session) error {
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		err := s.Get(&row, squirrel.Select("transaction_id, asset_code, amount, stellar_public_key, failure_count").
			From(transactionsQueueTableName).
			Where("pooled = false AND (locked_until is null OR locked_until < ?)"+
				" AND failure_count < ?", time.Now(), maxProcessingFailureCount).
			OrderBy("failure_count ASC, id ASC").
			Suffix("FOR UPDATE SKIP Locked").
			Limit(1))
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil
			}
			return errors.Wrap(err, "failed to get transaction from the queue")
		}

		// set processing lock
		sessionToken, err = newToken()
		if err != nil {
			return errors.Wrap(err, "failed to create session token")
		}
		where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode}
		_, err = transactionsQueueTable.Update(nil, where).
			Set("locked_until", time.Now().Add(defaultTransactionLockTTL)).
			Set("locked_token", sessionToken).
			Exec()
		return err
	})
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to find and lock transaction")
	case row.TransactionID == "": // transient
		return nil
	}
	// process callback without surrounding transaction
	transaction := row.toQueueTransaction()
	defer d.releaseTransactionLock(transaction.TransactionID, sessionToken)

	if err := transactionHandler(transaction); err != nil {
		if err != context.DeadlineExceeded {
			// increase failure counter
			_ = withTx(d.session, func(s *db.Session) error {
				where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode, "locked_token": sessionToken}
				transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
				_, err := transactionsQueueTable.Update(nil, where).Set("failure_count", row.FailureCount+1).Exec()
				return err
			})
		}
		return errors.Wrap(err, "failed to process transaction")
	}

	// update pooled status
	err = withTx(d.session, func(s *db.Session) error {
		where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode, "locked_token": sessionToken}
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		_, err := transactionsQueueTable.Update(nil, where).Set("pooled", true).Exec()
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to set pooled status for transaction")
	}
	return nil
}

func (d *PostgresDatabase) releaseTransactionLock(transactionID string, sessionToken string) error {
	return withTx(d.session, func(s *db.Session) error {
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		where := map[string]interface{}{"transaction_id": transactionID, "locked_token": sessionToken}
		_, err := transactionsQueueTable.Update(nil, where).
			Set("locked_until", nil).
			Set("locked_token", nil).
			Exec()
		return err
	})
}

func newToken() (string, error) {
	raw := make([]byte, 8)
	_, err := rand.Read(raw)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", raw), nil
}

var errRollbackTrigger = errors.New("rollback trigger")

func withTx(session *db.Session, f func(s *db.Session) error) error {
	newSession := session.Clone()
	if err := newSession.Begin(); err != nil {
		return errors.Wrap(err, "failed to start db transaction")
	}
	defer newSession.Rollback()

	if err := f(newSession); err != nil {
		// is reserved  error for flow control
		if err == errRollbackTrigger {
			return nil
		}
		return err
	}
	return newSession.Commit()
}

func readOnly(f func(s *db.Session) error) func(s *db.Session) error {
	return func(s *db.Session) error {
		if _, err := s.ExecRaw("SET transaction READ ONLY"); err != nil {
			return err
		}
		if err := f(s); err != nil {
			return err
		}
		return errRollbackTrigger
	}

}
