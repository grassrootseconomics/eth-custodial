package worker

import (
	"context"
	"log/slog"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/riverqueue/river"
)

type (
	AccountCreateArgs struct {
		TrackingId string      `json:"trackingId"`
		KeyPair    keypair.Key `json:"keypair"`
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

	if err := w.store.InsertKeyPair(ctx, tx, job.Args.KeyPair); err != nil {
		return err
	}

	return nil
}
