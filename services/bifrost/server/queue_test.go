package server

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/stellar"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
)

func TestPollTransactionQueueShouldRetryOnErrors(t *testing.T) {
	var counter uint64 = 0
	stub := func(func(queue.Transaction) error) (bool, error) {
		atomic.AddUint64(&counter, 1)
		return false, errors.New("test, please ignore")
	}
	server := Server{
		TransactionsQueue:          queuedTransactionStub(stub),
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	defaultQueueRetryDelay = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// when
	go server.poolTransactionsQueue(ctx)

	// then
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout before stub got called")
		default:
			if counter > 1 {
				return
			}
		}
	}
}

func TestPollTransactionQueueShouldNotSleepWhenQueueHasElements(t *testing.T) {
	var counter uint64 = 0
	stub := func(func(queue.Transaction) error) (bool, error) {
		atomic.AddUint64(&counter, 1)
		return true, nil
	}
	server := Server{
		TransactionsQueue:          queuedTransactionStub(stub),
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	defaultQueueRetryDelay = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// when
	go server.poolTransactionsQueue(ctx)

	// then
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout before stub got called")
		default:
			if counter > 1 {
				return
			}
		}
	}
}

func TestPollTransactionQueueShouldExitWhenCtxClosed(t *testing.T) {
	stub := func(func(queue.Transaction) error) (bool, error) {
		t.Fatal("unexpected call to transaction handler")
		return false, nil
	}
	server := Server{
		TransactionsQueue:          queuedTransactionStub(stub),
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	cancel()

	// when
	go server.poolTransactionsQueue(ctx)

	// then
	assert.Error(t, ctx.Err())
}

type queuedTransactionStub func(func(queue.Transaction) error) (bool, error)

func (s queuedTransactionStub) QueueAdd(_ queue.Transaction) error {
	return errors.New("not supported")
}
func (s queuedTransactionStub) WithQueuedTransaction(f func(queue.Transaction) error) (bool, error) {
	return s(f)
}
