package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/grassrootseconomics/eth-custodial/internal/gas"
	"github.com/grassrootseconomics/eth-custodial/internal/pub"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

type (
	WorkerOpts struct {
		MaxWorkers          int
		Registry            map[string]common.Address
		HealthCheckInterval time.Duration
		GasOracle           gas.GasOracle
		Store               store.Store
		Logg                *slog.Logger
		ChainProvider       *ethutils.Provider
		Pub                 *pub.Pub
	}

	signer struct {
		gasOracle     gas.GasOracle
		chainProvider *ethutils.Provider
	}

	WorkerContainer struct {
		Registry      map[string]common.Address
		GasOracle     gas.GasOracle
		Store         store.Store
		Logg          *slog.Logger
		Pub           *pub.Pub
		ChainProvider *ethutils.Provider
		QueueClient   *river.Client[pgx.Tx]
	}
)

const (
	migrationTimeout    = 15 * time.Second
	healthCheckInterval = 2 * time.Minute
)

func New(o WorkerOpts) (*WorkerContainer, error) {
	workerContainer := &WorkerContainer{
		Registry:      o.Registry,
		GasOracle:     o.GasOracle,
		Store:         o.Store,
		Logg:          o.Logg,
		Pub:           o.Pub,
		ChainProvider: o.ChainProvider,
		QueueClient:   nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	riverPgxDriver := riverpgxv5.New(o.Store.Pool())
	riverMigrator, err := rivermigrate.New(riverPgxDriver, &rivermigrate.Config{
		Logger: o.Logg,
	})
	if err != nil {
		return nil, err
	}

	_, err = riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return nil, err
	}

	workers, err := setupWorkers(workerContainer)
	if err != nil {
		return nil, err
	}

	workerContainer.QueueClient, err = river.NewClient(riverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {
				MaxWorkers: o.MaxWorkers,
			},
		},
		Workers:      workers,
		PeriodicJobs: setupHealthCheck(),
		Logger:       o.Logg,
	})

	return workerContainer, nil
}

func (w *WorkerContainer) Start(ctx context.Context) error {
	return w.QueueClient.Start(ctx)
}

func (w *WorkerContainer) Stop(ctx context.Context) error {
	w.Logg.Info("shutting down river queue")
	return w.QueueClient.Stop(ctx)
}

func setupWorkers(wc *WorkerContainer) (*river.Workers, error) {
	workers := river.NewWorkers()

	if err := river.AddWorkerSafely(workers, &TokenTransferWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &TokenSweepWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &AccountCreateWorker{wc: wc, custodialRegistrationProxy: wc.Registry[ethutils.CustodialProxy]}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &DisptachWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &DispatchHealthCheckWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &RetrierWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &PoolSwapWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &PoolDepositWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &GasRefillWorker{wc: wc, gasFaucet: wc.Registry[ethutils.GasFaucet]}); err != nil {
		return nil, err
	}

	return workers, nil
}

func setupHealthCheck() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(healthCheckInterval),
			func() (river.JobArgs, *river.InsertOpts) {
				return DispatchHealthCheckArgs{}, nil
			},
			&river.PeriodicJobOpts{
				RunOnStart: true,
			},
		),
	}
}
