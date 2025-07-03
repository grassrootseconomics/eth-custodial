package worker

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	custodialEvent "github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
	"github.com/riverqueue/river"
)

type (
	DispatchHealthCheckArgs struct{}

	DispatchHealthCheckWorker struct {
		river.WorkerDefaults[DispatchHealthCheckArgs]
		wc *WorkerContainer
	}
)

const DispatchHealthCheckID = "DISPATCH_HEALTHCHECK"

func (DispatchHealthCheckArgs) Kind() string { return DispatchHealthCheckID }

func (w *DispatchHealthCheckWorker) Work(ctx context.Context, _ *river.Job[DispatchHealthCheckArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	otx, err := w.wc.store.GetFailedOTX(ctx, tx)
	if err != nil {
		return err
	}

	if len(otx) < 1 {
		return nil
	}

	var txsToCheck []*store.OTX
	for _, v := range otx {
		if v.DispatchStatus == store.IN_NETWORK && time.Since(v.UpdatedAt) > time.Minute {
			txsToCheck = append(txsToCheck, v)
		}
	}

	if len(txsToCheck) > 0 {
		calls := make([]w3types.RPCCaller, len(txsToCheck))
		receipts := make([]*types.Receipt, len(txsToCheck))

		for i, v := range txsToCheck {
			calls[i] = eth.TxReceipt(common.HexToHash(v.TxHash)).Returns(&receipts[i])
		}
		w.wc.logg.Debug("calls size", "size", len(calls))

		var batchErr w3.CallErrors
		if err := w.wc.chainProvider.Client.CallCtx(ctx, calls...); errors.As(err, &batchErr) {
			for i, e := range batchErr {
				if e != nil {
					w.wc.logg.Warn("failed to fetch transaction receipts in batch", "call_pos", i, "error", e)
				}
			}
		}
		for i, v := range receipts {
			if v != nil && v.BlockNumber != nil {
				if v.Status == 1 {
					updateDispatchStatus := store.DispatchTx{
						OTXID:  txsToCheck[i].ID,
						Status: store.SUCCESS,
					}
					if err := w.wc.store.UpdateDispatchTxStatus(ctx, tx, updateDispatchStatus); err != nil {
						return err
					}
					w.wc.pub.Send(ctx, custodialEvent.Event{
						TrackingID: txsToCheck[i].TrackingID,
						Status:     updateDispatchStatus.Status,
					})

					w.wc.logg.Debug("health check manually updated otx status to SUCCESS",
						"otx_id", txsToCheck[i].ID,
						"tx_hash", v.TxHash.Hex(),
						"block_number", v.BlockNumber,
					)
				}

			}
		}
	}

	return tx.Commit(ctx)
}
