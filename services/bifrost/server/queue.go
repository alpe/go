package server

import (
	"context"
	"time"

	"github.com/stellar/go/services/bifrost/queue"
)

var defaultQueueRetryDelay = time.Second

const (
	maxProcessingTime = 2 * time.Minute
)

// poolTransactionsQueue pools transactions queue which contains only processed and
// validated transactions and sends it to StellarAccountConfigurator for account configuration.
func (s *Server) poolTransactionsQueue(ctx context.Context) {
	s.log.Info("Started pooling transactions queue")

	exit := ctx.Done()
	retryDelayer := time.NewTimer(defaultQueueRetryDelay)
	defer retryDelayer.Stop()
	for ctx.Err() == nil {
		if empty, err := s.TransactionsQueue.IsEmpty(); err != nil || empty {
			retryDelayer.Reset(maxProcessingTime)
			select {
			case <-exit:
				break
			case <-retryDelayer.C:
				continue
			}
		}
		go s.processNextQueuedTransaction(ctx)
	}
	s.log.Info("Stopped pooling transactions queue")
}

func (s *Server) processNextQueuedTransaction(parentCtx context.Context) {
	err := s.TransactionsQueue.WithQueuedTransaction(func(transaction queue.Transaction) error {
		ctx, done := context.WithTimeout(parentCtx, maxProcessingTime)
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
	if err != nil {
		s.log.WithField("err", err).Error("Error processing transactions queue")
	}
}
