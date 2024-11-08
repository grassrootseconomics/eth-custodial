package worker

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	TokenSweepArgs struct {
		TrackingID   string `json:"trackingId"`
		From         string `json:"from"`
		To           string `json:"to"`
		TokenAddress string `json:"tokenAddress"`
	}

	TokenSweepWorker struct {
		river.WorkerDefaults[TokenSweepArgs]
		wc *WorkerContainer
	}
)

func (TokenSweepArgs) Kind() string { return store.TOKEN_SWEEP }

func (w *TokenSweepWorker) Work(ctx context.Context, job *river.Job[TokenSweepArgs]) error {
	tx, err := w.wc.Store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := w.wc.Store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.wc.Store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	input, err := abi[Sweep].EncodeArgs(
		ethutils.HexToAddress(job.Args.To),
	)
	if err != nil {
		return err
	}

	gasSettings, err := w.wc.GasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.wc.ChainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.TokenAddress),
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
		OTXType:       store.TOKEN_SWEEP,
		SignerAccount: job.Args.From,
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
	w.wc.Pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

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