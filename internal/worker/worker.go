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
		// TODO: temporary patch for prod because poolIndex doesn't exist in the entry point registry
		Prod bool
	}

	signer struct {
		gasOracle     gas.GasOracle
		chainProvider *ethutils.Provider
	}

	WorkerContainer struct {
		queueClient   *river.Client[pgx.Tx]
		registry      map[string]common.Address
		gasOracle     gas.GasOracle
		store         store.Store
		logg          *slog.Logger
		pub           *pub.Pub
		chainProvider *ethutils.Provider
		prod          bool
	}
)

const (
	migrationTimeout    = 15 * time.Second
	healthCheckInterval = 2 * time.Minute
)

func New(o WorkerOpts) (*WorkerContainer, error) {
	workerContainer := &WorkerContainer{
		queueClient:   nil,
		registry:      o.Registry,
		gasOracle:     o.GasOracle,
		store:         o.Store,
		logg:          o.Logg,
		pub:           o.Pub,
		chainProvider: o.ChainProvider,
		prod:          o.Prod,
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

	workerContainer.queueClient, err = river.NewClient(riverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {
				MaxWorkers: o.MaxWorkers,
			},
		},
		Workers:      workers,
		PeriodicJobs: setupHealthCheck(),
		Logger:       o.Logg,
	})
	if err != nil {
		return nil, err
	}

	return workerContainer, nil
}

func (w *WorkerContainer) Start(ctx context.Context) error {
	return w.queueClient.Start(ctx)
}

func (w *WorkerContainer) Stop(ctx context.Context) error {
	w.logg.Info("shutting down river queue")
	return w.queueClient.Stop(ctx)
}

func (w *WorkerContainer) Client() *river.Client[pgx.Tx] {
	return w.queueClient
}

func setupWorkers(wc *WorkerContainer) (*river.Workers, error) {
	workers := river.NewWorkers()

	if err := river.AddWorkerSafely(workers, &TokenTransferWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &TokenSweepWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &AccountCreateWorker{wc: wc, custodialRegistrationProxy: wc.registry[ethutils.CustodialProxy]}); err != nil {
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

	if err := river.AddWorkerSafely(workers, &GasRefillWorker{wc: wc, gasFaucet: wc.registry[ethutils.GasFaucet]}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &GenericSignWorker{wc: wc}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &TokenDeployWorker{wc: wc, tokenIndex: wc.registry[ethutils.TokenIndex]}); err != nil {
		return nil, err
	}

	if err := river.AddWorkerSafely(workers, &PoolDeployWorker{wc: wc}); err != nil {
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
