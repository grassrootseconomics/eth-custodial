package worker

import (
	"context"

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
	tx, err := w.wc.Store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	otx, err := w.wc.Store.GetFailedOTX(ctx, tx)
	if err != nil {
		return err
	}

	if len(otx) < 1 {
		w.wc.Logg.Debug("dispatch health check: no failed otx found")
		return nil
	}

	for _, v := range otx {
		w.wc.Logg.Warn("dispatch health check: found failed otx", "otx_id", v.ID, "tracking_id", v.TrackingID, "account", v.SignerAccount, "status", v.DispatchStatus)
	}

	return nil
}
