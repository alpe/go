package stellar

import (
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// NewAccountXLMBalance is amount of lumens sent to new accounts
const NewAccountXLMBalance = "41"

func (ac *AccountConfigurator) Start() error {
	ac.log = common.CreateLogger("StellarAccountConfigurator")
	ac.log.Info("StellarAccountConfigurator starting")

	kp, err := keypair.Parse(ac.IssuerPublicKey)
	if err != nil || (err == nil && ac.IssuerPublicKey[0] != 'G') {
		err = errors.Wrap(err, "Invalid IssuerPublicKey")
		ac.log.Error(err)
		return err
	}

	kp, err = keypair.Parse(ac.SignerSecretKey)
	if err != nil || (err == nil && ac.SignerSecretKey[0] != 'S') {
		err = errors.Wrap(err, "Invalid SignerSecretKey")
		ac.log.Error(err)
		return err
	}

	ac.signerPublicKey = kp.Address()

	root, err := ac.Horizon.Root()
	if err != nil {
		err = errors.Wrap(err, "Error loading Horizon root")
		ac.log.Error(err)
		return err
	}

	if root.NetworkPassphrase != ac.NetworkPassphrase {
		return errors.Errorf("Invalid network passphrase (have=%s, want=%s)", root.NetworkPassphrase, ac.NetworkPassphrase)
	}

	err = ac.updateSequence()
	if err != nil {
		err = errors.Wrap(err, "Error loading issuer sequence number")
		ac.log.Error(err)
		return err
	}

	go ac.logStats()
	return nil
}

func (ac *AccountConfigurator) logStats() {
	for {
		ac.log.WithField("currently_processing", ac.processingCount).Info("Stats")
		time.Sleep(15 * time.Second)
	}
}

// ConfigureAccount configures a new account that participated in ICO.
// * First it creates a new account.
// * Once a trust line exists, it credits it with received number of ETH or BTC.
func (ac *AccountConfigurator) ConfigureAccount(destination, assetCode, amount string) error {
	localLog := ac.log.WithFields(log.F{
		"destination": destination,
		"assetCode":   assetCode,
		"amount":      amount,
	})
	localLog.Info("Configuring Stellar account")

	ac.processingCountMutex.Lock()
	ac.processingCount++
	ac.processingCountMutex.Unlock()

	defer func() {
		ac.processingCountMutex.Lock()
		ac.processingCount--
		ac.processingCountMutex.Unlock()
	}()

	destAccount, exists, err := ac.getAccount(destination)
	if err != nil {
		return errors.Wrap(err, "failed to load account from Horizon")
	}

	if !exists {
		localLog.WithField("destination", destination).Info("Creating Stellar account")
		if err := ac.createAccount(destination); err != nil {
			return errors.Wrap(err, "failed to create Stellar account")
		}

		if ac.OnAccountCreated != nil {
			ac.OnAccountCreated(destination)
		}
	}

	if !ac.trustlineExists(destAccount, assetCode) {
		// Wait for trust line to be created...
		for i := 0; i < 10; i++ {
			destAccount, err = ac.Horizon.LoadAccount(destination)
			if err != nil {
				return errors.Wrap(err, "failed to load account to check trust line")
			}

			if ac.trustlineExists(destAccount, assetCode) {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	localLog.Info("Trust line found")

	// When trust line found check if needs to authorize, then send token
	if ac.NeedsAuthorize {
		localLog.Info("Authorizing trust line")
		if err := ac.allowTrust(destination, assetCode, ac.TokenAssetCode); err != nil {
			return errors.Wrap(err, "authorizing trust line failed")
		}
	}

	localLog.Info("Sending token")
	if err := ac.sendToken(destination, assetCode, amount); err != nil {
		return errors.Wrap(err, "sending asset to account failed")
	}

	if ac.OnAccountCredited != nil {
		ac.OnAccountCredited(destination, assetCode, amount)
	}

	localLog.Info("Account successfully configured")
	return nil
}

func (ac *AccountConfigurator) getAccount(account string) (horizon.Account, bool, error) {
	var hAccount horizon.Account
	hAccount, err := ac.Horizon.LoadAccount(account)
	if err != nil {
		if err, ok := err.(*horizon.Error); ok && err.Response.StatusCode == http.StatusNotFound {
			return hAccount, false, nil
		}
		return hAccount, false, err
	}

	return hAccount, true, nil
}

func (ac *AccountConfigurator) trustlineExists(account horizon.Account, assetCode string) bool {
	for _, balance := range account.Balances {
		if balance.Asset.Issuer == ac.IssuerPublicKey && balance.Asset.Code == assetCode {
			return true
		}
	}

	return false
}
