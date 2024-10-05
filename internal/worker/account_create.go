package worker

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	AccountCreateArgs struct {
		TrackingID string      `json:"trackingId"`
		KeyPair    keypair.Key `json:"keypair"`
	}

	AccountCreateWorker struct {
		river.WorkerDefaults[AccountCreateArgs]
		wc *WorkerContainer
	}
)

const AccountCreateID = "ACCOUNT_CREATE"

func (AccountCreateArgs) Kind() string { return AccountCreateID }

func (w *AccountCreateWorker) Work(ctx context.Context, job *river.Job[AccountCreateArgs]) error {
	tx, err := w.wc.Store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := w.wc.Store.InsertKeyPair(ctx, tx, job.Args.KeyPair); err != nil {
		return err
	}

	systemKeypair, err := w.wc.Store.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(systemKeypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.wc.Store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	input, err := abi[Register].EncodeArgs(ethutils.HexToAddress(job.Args.KeyPair.Public))
	if err != nil {
		return err
	}

	gasSettings, err := w.wc.GasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.wc.ChainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
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

	rawTxHex := hexutil.Encode(rawTx)

	otxID, err := w.wc.Store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.ACCOUNT_REGISTER,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTxHex,
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.Store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = w.wc.QueueClient.InsertTx(ctx, tx, DispatchArgs{
		TrackingID: job.Args.TrackingID,
		OTXID:      otxID,
		RawTx:      rawTxHex,
	}, nil)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
