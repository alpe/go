package server

import (
	"context"
	"sync"
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
	stub := func(func(queue.Transaction) error) error {
		atomic.AddUint64(&counter, 1)
		return errors.New("test, please ignore")
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
	stub := func(func(queue.Transaction) error) error {
		atomic.AddUint64(&counter, 1)
		return nil
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
	stub := func(func(queue.Transaction) error) error {
		t.Fatal("unexpected call to transaction handler")
		return nil
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

func TestPollTransactionQueueShouldNotBlockWhileProcessing(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	var counter uint64 = 0
	stub := func(func(queue.Transaction) error) error {
		atomic.AddUint64(&counter, 1)
		wg.Wait()
		return nil
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

	// then test should not timeout
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout before stub got called")
		default:
			if counter > 1 {
				wg.Done()
				return
			}
		}
	}

}

type queuedTransactionStub func(func(queue.Transaction) error) error

func (s queuedTransactionStub) QueueAdd(_ queue.Transaction) error {
	return errors.New("not supported")
}
func (s queuedTransactionStub) WithQueuedTransaction(f func(queue.Transaction) error) error {
	return s(f)
}
func (s queuedTransactionStub) IsEmpty() (bool, error) {
	return false, nil
}
