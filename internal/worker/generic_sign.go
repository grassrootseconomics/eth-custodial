package worker

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	GenericSignArgs struct {
		TrackingID string `json:"trackingId"`
		To         string `json:"to"`
		From       string `json:"from"`
		Value      string `json:"value"`
		Data       string `json:"data"`
	}

	GenericSignWorker struct {
		river.WorkerDefaults[GenericSignArgs]
		wc *WorkerContainer
	}
)

func (GenericSignArgs) Kind() string { return store.GENERIC_SIGN }

func (w *GenericSignWorker) Work(ctx context.Context, job *river.Job[GenericSignArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := w.wc.store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.wc.store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	value, err := StringToBigInt(job.Args.Value, false)
	if err != nil {
		return err
	}

	to := ethutils.HexToAddress(job.Args.To)

	gasSettings, err := w.wc.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := types.SignNewTx(privateKey, w.wc.chainProvider.Signer, &types.DynamicFeeTx{
		Value:     value,
		To:        &to,
		Nonce:     nonce,
		Data:      common.FromHex(job.Args.Data),
		Gas:       gasSettings.GasLimit,
		GasFeeCap: gasSettings.GasFeeCap,
		GasTipCap: gasSettings.GasTipCap,
	})
	if err != nil {
		return err
	}

	rawTx, err := builtTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTxHex := hexutil.Encode(rawTx)

	otxID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.GENERIC_SIGN,
		SignerAccount: job.Args.From,
		RawTx:         rawTxHex,
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = w.wc.queueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      otxID,
				RawTx:      rawTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: GasRefillArgs{
				TrackingID: job.Args.TrackingID,
				Address:    job.Args.From,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 2,
			},
		},
	})
	if err != nil {
		return err
	}

	w.wc.pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

	return tx.Commit(ctx)
}
