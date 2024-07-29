package worker

import (
	"context"
	"log/slog"

	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/riverqueue/river"
)

type (
	AccountCreateArgs struct {
		TrackingId string `json:"trackingId"`
		PublicKey  string `json:"from"`
		PrivateKey string `json:"to"`
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
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	return nil
}
