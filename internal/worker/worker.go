package worker

import (
	"log/slog"

	"github.com/grassrootseconomics/celo-custodial/internal/gas"
	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	WorkerOpts struct {
		ChainID   int64
		GasOracle gas.GasOracle
		Store     store.Store
		Logg      *slog.Logger
	}

	signer struct {
		gasOracle     gas.GasOracle
		chainProvider *ethutils.Provider
	}
)

func New(o WorkerOpts) (*river.Workers, error) {
	signer := &signer{
		gasOracle:     o.GasOracle,
		chainProvider: ethutils.NewProvider("https://offline.only", o.ChainID),
	}

	workers := river.NewWorkers()

	if err := river.AddWorkerSafely(workers, &TokenTransferWorker{
		store:  o.Store,
		logg:   o.Logg,
		signer: signer,
	}); err != nil {
		return nil, err
	}

	return workers, nil
}
