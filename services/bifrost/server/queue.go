package server

import (
	"context"
	"time"

	"github.com/stellar/go/services/bifrost/queue"
)

var defaultQueueRetryDelay = time.Second
var maxProxessingTime = 2 * time.Minute

// poolTransactionsQueue pools transactions queue which contains only processed and
// validated transactions and sends it to StellarAccountConfigurator for account configuration.
func (s *Server) poolTransactionsQueue(parentCtx context.Context) {
	s.log.Info("Started pooling transactions queue")
	exit := parentCtx.Done()

	for {
		select {
		case <-exit:
			return
		default:
		}

		exists, err := s.TransactionsQueue.WithQueuedTransaction(func(transaction queue.Transaction) error {
			ctx, done := context.WithTimeout(parentCtx, maxProxessingTime)
			defer done()
			// blocking execution due to exclusive lock on transaction table
			s.log.WithField("transaction", transaction).Info("Received transaction from transactions queue")
			return s.StellarAccountConfigurator.ConfigureAccount(
				ctx,
				transaction.TransactionID,
				transaction.StellarPublicKey,
				string(transaction.AssetCode),
				transaction.Amount,
			)
		})
		switch {
		case err != nil:
			s.log.WithField("err", err).Error("Error processing transactions queue")
		case exists:
			continue
		}

		time.Sleep(defaultQueueRetryDelay)
	}
}
