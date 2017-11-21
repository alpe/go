package stellar

import (
	"strconv"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (ac *AccountConfigurator) createAccount(transactionID, destination string) error {
	xdr, err := ac.buildTransaction(
		build.CreateAccount(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Destination{destination},
			build.NativeAmount{NewAccountXLMBalance},
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to build transaction")
	}
	if err := ac.submissionArchive.Store(transactionID, SubmissionTypeCreateAccount, xdr); err != nil {
		return errors.Wrap(err, "failed to archive xdr")
	}
	if err = ac.submitXDR(xdr); err != nil {
		if _, ok := err.(*horizon.Error); ok {
			_ = ac.submissionArchive.Delete(transactionID, SubmissionTypeCreateAccount)
		}
		return err
	}
	return nil
}

func (ac *AccountConfigurator) allowTrust(trustor, assetCode, tokenAssetCode string) error {
	err := ac.submitTransaction(
		// Chain token received (BTC/ETH)
		build.AllowTrust(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Trustor{trustor},
			build.AllowTrustAsset{assetCode},
			build.Authorize{true},
		),
		// Destination token
		build.AllowTrust(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Trustor{trustor},
			build.AllowTrustAsset{tokenAssetCode},
			build.Authorize{true},
		),
	)
	if err != nil {
		return errors.Wrap(err, "Error submitting transaction")
	}

	return nil
}

func (ac *AccountConfigurator) sendToken(transactionID, destination, assetCode, amount string) error {
	xdr, err := ac.buildTransaction(
		build.Payment(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Destination{destination},
			build.CreditAmount{
				Code:   assetCode,
				Issuer: ac.IssuerPublicKey,
				Amount: amount,
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to build transaction")
	}
	if err := ac.submissionArchive.Store(transactionID, SubmissionTypeSendTokens, xdr); err != nil {
		return errors.Wrap(err, "failed to archive xdr")
	}
	if err := ac.submitXDR(xdr); err != nil {
		if _, ok := err.(*horizon.Error); ok {
			_ = ac.submissionArchive.Delete(transactionID, SubmissionTypeSendTokens)
		}
		return err
	}
	return nil
}

func (ac *AccountConfigurator) submitTransaction(mutators ...build.TransactionMutator) error {
	tx, err := ac.buildTransaction(mutators...)
	if err != nil {
		return errors.Wrap(err, "Error building transaction")
	}
	return ac.submitXDR(tx)
}

func (ac *AccountConfigurator) submitXDR(xdr string) error {
	localLog := log.WithField("tx", xdr)
	localLog.Info("Submitting transaction")

	if _, err := ac.Horizon.SubmitTransaction(xdr); err != nil {
		fields := log.F{"err": err}
		if err, ok := err.(*horizon.Error); ok {
			fields["result"] = string(err.Problem.Extras["result_xdr"])
			ac.updateSequence()
		}
		return errors.Wrap(err, "Error submitting transaction")
	}

	localLog.Info("Transaction successfully submitted")
	return nil
}

func (ac *AccountConfigurator) buildTransaction(mutators ...build.TransactionMutator) (string, error) {
	muts := []build.TransactionMutator{
		build.SourceAccount{ac.signerPublicKey},
		build.Sequence{ac.getSequence()},
		build.Network{ac.NetworkPassphrase},
	}
	muts = append(muts, mutators...)
	tx := build.Transaction(muts...)
	if tx.Err != nil {
		return "", tx.Err
	}
	txe := tx.Sign(ac.SignerSecretKey)
	return txe.Base64()
}

func (ac *AccountConfigurator) updateSequence() error {
	ac.sequenceMutex.Lock()
	defer ac.sequenceMutex.Unlock()

	account, err := ac.Horizon.LoadAccount(ac.signerPublicKey)
	if err != nil {
		err = errors.Wrap(err, "Error loading issuing account")
		ac.log.Error(err)
		return err
	}

	ac.sequence, err = strconv.ParseUint(account.Sequence, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "Invalid account.Sequence")
		ac.log.Error(err)
		return err
	}

	return nil
}

func (ac *AccountConfigurator) getSequence() uint64 {
	ac.sequenceMutex.Lock()
	defer ac.sequenceMutex.Unlock()
	ac.sequence++
	sequence := ac.sequence
	return sequence
}
