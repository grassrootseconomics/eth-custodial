package worker

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/lmittmann/w3/module/eth"
	"github.com/riverqueue/river"
)

type (
	DispatchArgs struct {
		TrackingID string `json:"trackingId"`
		OTXID      uint64 `json:"otxId"`
		RawTx      string `json:"rawTx"`
	}

	DisptachWorker struct {
		river.WorkerDefaults[DispatchArgs]
		wc *WorkerContainer
	}
)

const DispatchID = "DISPATCH"

func (DispatchArgs) Kind() string { return DispatchID }

func (w *DisptachWorker) Work(ctx context.Context, job *river.Job[DispatchArgs]) error {
	tx, err := w.wc.Store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rawTx, err := hexutil.Decode(job.Args.RawTx)
	if err != nil {
		return err
	}

	if err := w.sendRawTx(ctx, rawTx); err != nil {
		if err := w.wc.Store.UpdateDispatchTxStatus(ctx, tx, store.DispatchTx{
			OTXID:  job.Args.OTXID,
			Status: store.UNKNOWN_RPC_ERROR,
		}); err != nil {
			return err
		}

		return err
	}
	if err := w.wc.Store.UpdateDispatchTxStatus(ctx, tx, store.DispatchTx{
		OTXID:  job.Args.OTXID,
		Status: store.IN_NETWORK,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (w *DisptachWorker) sendRawTx(ctx context.Context, rawTx []byte) error {
	var txHash common.Hash
	sendRawTxCall := eth.SendRawTx(rawTx).Returns(&txHash)
	if err := w.wc.ChainProvider.Client.CallCtx(ctx, sendRawTxCall); err != nil {
		return err
	}

	return nil
}
