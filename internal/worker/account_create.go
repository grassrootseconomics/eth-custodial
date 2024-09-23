package worker

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	AccountCreateArgs struct {
		TrackingId string      `json:"trackingId"`
		KeyPair    keypair.Key `json:"keypair"`
	}

	AccountCreateWorker struct {
		river.WorkerDefaults[AccountCreateArgs]
		store  store.Store
		logg   *slog.Logger
		signer *signer
	}
)

const AccountCreateID = "ACCOUNT_CREATE"

func (AccountCreateArgs) Kind() string { return AccountCreateID }

func (w *AccountCreateWorker) Work(ctx context.Context, job *river.Job[AccountCreateArgs]) error {
	tx, err := w.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := w.store.InsertKeyPair(ctx, tx, job.Args.KeyPair); err != nil {
		return err
	}

	systemKeypair, err := w.store.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(systemKeypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	input, err := abi[Register].EncodeArgs(ethutils.HexToAddress(job.Args.KeyPair.Public))
	if err != nil {
		return err
	}

	gasSettings, err := w.signer.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.signer.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(custodialRegistrationProxyAddress),
		InputData:       input,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           nonce,
	})
	if err != nil {
		return err
	}

	rawTx, err := builtTx.MarshalBinary()
	if err != nil {
		return err
	}

	if err := w.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingId,
		OTXType:       store.ACCOUNT_REGISTER,
		SignerAccount: systemKeypair.Public,
		RawTx:         hexutil.Encode(rawTx),
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
