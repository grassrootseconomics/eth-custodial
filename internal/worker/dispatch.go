package worker

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/lmittmann/w3"
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

	updateTxStatus := store.DispatchTx{
		OTXID:  job.Args.OTXID,
		Status: store.UNKNOWN_RPC_ERROR,
	}
	if err := w.sendRawTx(ctx, rawTx); err != nil {
		dispatchErr, ok := err.(*DispatchError)
		if ok {
			if dispatchErr.Err == ErrNetwork {
				updateTxStatus.Status = store.NETWORK_ERROR
				w.wc.Logg.Error("network related dispatch error", "original_error", dispatchErr.OriginalErr)
			} else {
				w.wc.Logg.Error("chain related dispatch error", "error", dispatchErr.Err, "original_error", dispatchErr.OriginalErr)
				switch dispatchErr.Err {
				case ErrGasPriceTooLow:
					updateTxStatus.Status = store.LOW_GAS_PRICE
				case ErrInsufficientGas:
					updateTxStatus.Status = store.NO_GAS
				case ErrNonceTooLow:
					updateTxStatus.Status = store.LOW_NONCE
				case ErrReplacementTxUnderpriced:
					updateTxStatus.Status = store.REPLACEMENT_UNDERPRICED
				}
			}
		}
		if err := w.wc.Store.UpdateDispatchTxStatus(ctx, tx, updateTxStatus); err != nil {
			return err
		}
		w.wc.Logg.Error("dispatch error", "error", err)
		return err
	}
	updateTxStatus.Status = store.IN_NETWORK

	if err := w.wc.Store.UpdateDispatchTxStatus(ctx, tx, updateTxStatus); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (w *DisptachWorker) sendRawTx(ctx context.Context, rawTx []byte) error {
	var txHash common.Hash
	sendRawTxCall := eth.SendRawTx(rawTx).Returns(&txHash)

	var errs w3.CallErrors
	if err := w.wc.ChainProvider.Client.CallCtx(ctx, sendRawTxCall); errors.As(err, &errs) {
		jsonErr, ok := errs[0].(rpc.Error)
		if ok {
			if jsonRPCError := handleJSONRPCError(jsonErr.Error()); jsonRPCError != nil {
				return &DispatchError{
					Err:         jsonRPCError,
					OriginalErr: jsonErr,
				}
			}
		}
	} else if err != nil {
		return handleNetworkError(err)
	}

	return nil
}
