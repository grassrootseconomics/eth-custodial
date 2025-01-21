package worker

import (
	"context"
	"fmt"

	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/riverqueue/river"
)

type (
	RetrierArgs struct {
		TrackingID string `json:"trackingId"`
	}

	RetrierWorker struct {
		river.WorkerDefaults[RetrierArgs]
		wc *WorkerContainer
	}
)

const RetrierID = "RETRIER"

func (RetrierArgs) Kind() string { return RetrierID }

func (w *RetrierWorker) Work(ctx context.Context, job *river.Job[RetrierArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	otx, err := w.wc.store.GetOTXByTrackingID(ctx, tx, job.Args.TrackingID)
	if err != nil {
		return err
	}

	if len(otx) < 1 {
		return fmt.Errorf("retrier: no otx found for tracking id %s", job.Args.TrackingID)
	}

	for _, v := range otx {
		if !isChainError(v.DispatchStatus) {
			w.wc.logg.Debug("retrier: skipping non-chain error withing otx chain sequence", "otx_id", v.ID, "status", v.DispatchStatus)
			break
		}

		switch v.DispatchStatus {
		case store.NO_GAS:
			w.wc.logg.Warn("retrier: encountered NO_GAS error during dispatch attempting to topup account", "account", v.SignerAccount)
			return nil
		case store.LOW_GAS_PRICE, store.REPLACEMENT_UNDERPRICED:
			w.wc.logg.Warn("retrier: encountered low gas error during dispatch attempting to bump gas", "account", v.SignerAccount, "reason", v.DispatchStatus)
			break
		case store.LOW_NONCE:
			w.wc.logg.Error("retrier: encountered low nonce error during dispatch", "account", v.SignerAccount)
			return nil
		}
	}

	return nil
}

func isChainError(status string) bool {
	switch status {
	case store.NO_GAS, store.LOW_GAS_PRICE, store.REPLACEMENT_UNDERPRICED, store.LOW_NONCE:
		return true
	}
	return false
}
