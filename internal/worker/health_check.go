package worker

import (
	"context"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/store"
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
		w.wc.logg.Debug("dispatch health check: no failed otx found")
		return nil
	}

	for _, v := range otx {
		if v.DispatchStatus == store.IN_NETWORK {
			if v.UpdatedAt.Sub(time.Now()) < time.Minute*1 {
				break
			}
		}
		w.wc.logg.Warn("dispatch health check: found failed otx", "otx_id", v.ID, "tracking_id", v.TrackingID, "account", v.SignerAccount, "status", v.DispatchStatus)
	}

	return nil
}
