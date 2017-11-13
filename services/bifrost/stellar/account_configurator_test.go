package stellar

import (
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigureAccount(t *testing.T) {
	myIssuersKeyPair, _ := keypair.Random()
	mySignersKeyPair, _ := keypair.Random()
	myReceiversKeyPair, _ := keypair.Random()
	horizonMock := &horizon.MockClient{}
	ac := &AccountConfigurator{
		Horizon:           horizonMock,
		IssuerPublicKey:   myIssuersKeyPair.Address(),
		signerPublicKey:   mySignersKeyPair.Address(),
		SignerSecretKey:   mySignersKeyPair.Seed(),
		NeedsAuthorize:    false,
		NetworkPassphrase: network.TestNetworkPassphrase,
		log:               common.CreateLogger("test account configurer"),
	}

	horizonMock.Mock.
		On("LoadAccount", myReceiversKeyPair.Address()).
		Return(horizon.Account{
			Balances: []horizon.Balance{
				{Asset: horizon.Asset{Issuer: myIssuersKeyPair.Address(), Code: "myAssetCode"}},
			},
		}, nil)
	horizonMock.Mock.
		On("SubmitTransaction", mock.Anything).
		Return(horizon.TransactionSuccess{}, nil)

	// when
	err := ac.ConfigureAccount(myReceiversKeyPair.Address(), "myAssetCode", "1")

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
}
