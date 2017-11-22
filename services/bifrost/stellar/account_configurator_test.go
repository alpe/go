package stellar

import (
	"context"
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
		submissionArchive: &devNullArchiver{},
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
	err := ac.ConfigureAccount(context.Background(), "myTxID", myReceiversKeyPair.Address(), "myAssetCode", "1")

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
}

type devNullArchiver struct{}

func (d *devNullArchiver) Find(txID, assetCode string, st SubmissionType) (string, error) {
	return "", nil
}
func (d *devNullArchiver) Store(txID, assetCode string, st SubmissionType, xdr string) error {
	return nil
}
func (d *devNullArchiver) Delete(txID, assetCode string, st SubmissionType) error {
	return nil
}
