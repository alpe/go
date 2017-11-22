package stellar

import (
	"context"
	"net/http"
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigureAccountShortPath(t *testing.T) {
	myIssuersKeyPair, _ := keypair.Random()
	myReceiversKeyPair, _ := keypair.Random()
	mySignersKeyPair, _ := keypair.Random()
	horizonMock := &horizon.MockClient{}
	archiver := &MockArchiver{}
	ac := &AccountConfigurator{
		IssuerPublicKey:   myIssuersKeyPair.Address(),
		signerPublicKey:   mySignersKeyPair.Address(),
		SignerSecretKey:   mySignersKeyPair.Seed(),
		NeedsAuthorize:    false,
		NetworkPassphrase: network.TestNetworkPassphrase,
		log:               common.CreateLogger("test account configurer"),
		submissionArchive: archiver,
		Horizon:           horizonMock,
	}
	horizonMock.Mock.
		On("LoadAccount", myReceiversKeyPair.Address()).
		Return(horizon.Account{
			Balances: []horizon.Balance{
				{Asset: horizon.Asset{Issuer: myIssuersKeyPair.Address(), Code: "myAssetCode"}},
			},
		}, nil).
		On("SubmitTransaction", mock.Anything).
		Return(horizon.TransactionSuccess{}, nil)

	archiver.On("Find", "myTxID", "myAssetCode", SubmissionTypeSendTokens).
		Return("", nil).
		On("Store", "myTxID", "myAssetCode", SubmissionTypeSendTokens, mock.Anything).
		Return(nil)

	// when
	err := ac.ConfigureAccount(context.Background(), "myTxID", myReceiversKeyPair.Address(), "myAssetCode", "1")

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
	archiver.Mock.AssertExpectations(t)
}

func TestConfigureAccountFullPath(t *testing.T) {
	myIssuersKeyPair, _ := keypair.Random()
	myReceiversKeyPair, _ := keypair.Random()
	mySignersKeyPair, _ := keypair.Random()
	horizonMock := &horizon.MockClient{}
	archiver := &MockArchiver{}
	ac := &AccountConfigurator{
		IssuerPublicKey:   myIssuersKeyPair.Address(),
		signerPublicKey:   mySignersKeyPair.Address(),
		SignerSecretKey:   mySignersKeyPair.Seed(),
		TokenAssetCode:    "myTokenAsset",
		NeedsAuthorize:    true,
		NetworkPassphrase: network.TestNetworkPassphrase,
		log:               common.CreateLogger("test account configurer"),
		submissionArchive: archiver,
		Horizon:           horizonMock,
	}
	horizonMock.Mock.
		On("LoadAccount", myReceiversKeyPair.Address()).
		Return(horizon.Account{}, &horizon.Error{Response: &http.Response{StatusCode: http.StatusNotFound}}).Once()

	horizonMock.Mock.
		On("SubmitTransaction", mock.Anything). // create account
		Return(horizon.TransactionSuccess{}, nil).
		On("LoadAccount", myReceiversKeyPair.Address()).
		Return(horizon.Account{ // trust line exists now
			Balances: []horizon.Balance{
				{Asset: horizon.Asset{Issuer: myIssuersKeyPair.Address(), Code: "myAssetCode"}},
			},
		}, nil).
		On("SubmitTransaction", mock.Anything). // allow trust
		Return(horizon.TransactionSuccess{}, nil).
		On("SubmitTransaction", mock.Anything). // send tokens
		Return(horizon.TransactionSuccess{}, nil)

	archiver.On("Find", "myTxID", "myAssetCode", SubmissionTypeCreateAccount).
		Return("", nil).
		On("Store", "myTxID", "myAssetCode", SubmissionTypeCreateAccount, mock.Anything).
		Return(nil).
		On("Find", "myTxID", "myAssetCode", SubmissionTypeSendTokens).
		Return("", nil).
		On("Store", "myTxID", "myAssetCode", SubmissionTypeSendTokens, mock.Anything).
		Return(nil)

	// when
	err := ac.ConfigureAccount(context.Background(), "myTxID", myReceiversKeyPair.Address(), "myAssetCode", "1")

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
	archiver.Mock.AssertExpectations(t)
}

func TestShouldUseArchivedXDRForCreateAccountWhenExists(t *testing.T) {
	myIssuersKeyPair, _ := keypair.Random()
	myReceiversKeyPair, _ := keypair.Random()
	mySignersKeyPair, _ := keypair.Random()

	horizonMock := &horizon.MockClient{}
	archiver := &MockArchiver{}
	ac := &AccountConfigurator{
		IssuerPublicKey:   myIssuersKeyPair.Address(),
		signerPublicKey:   mySignersKeyPair.Address(),
		SignerSecretKey:   mySignersKeyPair.Seed(),
		TokenAssetCode:    "myTokenAsset",
		NeedsAuthorize:    true,
		NetworkPassphrase: network.TestNetworkPassphrase,
		log:               common.CreateLogger("test account configurer"),
		submissionArchive: archiver,
		Horizon:           horizonMock,
	}
	archiver.On("Find", "myTxID", "myAssetCode", SubmissionTypeCreateAccount).
		Return("myXDRContent", nil)

	horizonMock.Mock.
		On("SubmitTransaction", "myXDRContent"). // create account
		Return(horizon.TransactionSuccess{}, nil)

	// when
	err := ac.doCreateAccount("myTxID", "myAssetCode", myReceiversKeyPair.Address())

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
	archiver.Mock.AssertExpectations(t)
}
func TestShouldUseArchivedXDRFoSendTokensWhenExists(t *testing.T) {
	myIssuersKeyPair, _ := keypair.Random()
	myReceiversKeyPair, _ := keypair.Random()
	mySignersKeyPair, _ := keypair.Random()

	horizonMock := &horizon.MockClient{}
	archiver := &MockArchiver{}
	ac := &AccountConfigurator{
		IssuerPublicKey:   myIssuersKeyPair.Address(),
		signerPublicKey:   mySignersKeyPair.Address(),
		SignerSecretKey:   mySignersKeyPair.Seed(),
		TokenAssetCode:    "myTokenAsset",
		NeedsAuthorize:    true,
		NetworkPassphrase: network.TestNetworkPassphrase,
		log:               common.CreateLogger("test account configurer"),
		submissionArchive: archiver,
		Horizon:           horizonMock,
	}
	archiver.On("Find", "myTxID", "myAssetCode", SubmissionTypeSendTokens).
		Return("myXDRContent", nil)

	horizonMock.Mock.
		On("SubmitTransaction", "myXDRContent"). // create account
		Return(horizon.TransactionSuccess{}, nil)

	// when
	err := ac.doSendTokens(context.Background(), "myTxID", "myAssetCode", myReceiversKeyPair.Address(), "100")

	// then
	require.NoError(t, err)
	horizonMock.Mock.AssertExpectations(t)
	archiver.Mock.AssertExpectations(t)
}

type MockArchiver struct {
	mock.Mock
}

func (m *MockArchiver) Find(txID, assetCode string, st SubmissionType) (string, error) {
	a := m.Called(txID, assetCode, st)
	return a.Get(0).(string), a.Error(1)
}

func (m *MockArchiver) Store(txID, assetCode string, st SubmissionType, xdr string) error {
	a := m.Called(txID, assetCode, st, xdr)
	return a.Error(0)
}

func (m *MockArchiver) Delete(txID, assetCode string, st SubmissionType) error {
	a := m.Called(txID, assetCode, st)
	return a.Error(0)
}
