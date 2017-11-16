package server

import (
	"context"
	"math/big"
	"net/http"

	"github.com/stellar/go/services/bifrost/bitcoin"
	"github.com/stellar/go/services/bifrost/config"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/ethereum"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/sse"
	"github.com/stellar/go/services/bifrost/stellar"
	"github.com/stellar/go/support/log"
)

type Server struct {
	BitcoinListener            *bitcoin.Listener            `inject:""`
	BitcoinAddressGenerator    *bitcoin.AddressGenerator    `inject:""`
	Config                     *config.Config               `inject:""`
	Database                   database.Database            `inject:""`
	EthereumListener           *ethereum.Listener           `inject:""`
	EthereumAddressGenerator   *ethereum.AddressGenerator   `inject:""`
	StellarAccountConfigurator *stellar.AccountConfigurator `inject:""`
	TransactionsQueue          queue.Queue                  `inject:""`
	SSEServer                  sse.ServerInterface          `inject:""`

	MinimumValueBtc string
	MinimumValueEth string

	minimumValueSat            int64
	minimumValueWei            *big.Int
	httpServer                 *http.Server
	log                        *log.Entry
	stopTransactionQueueWorker context.CancelFunc
}

type GenerateAddressResponse struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}
