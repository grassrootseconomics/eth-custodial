package worker

import (
	"log/slog"

	"github.com/grassrootseconomics/celo-custodial/internal/gas"
	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
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
		store         store.Store
		logg          *slog.Logger
		chainProvider *ethutils.Provider
	}
)

func New(o WorkerOpts) {
	// bootstrap inidividal workers
	_ = &signer{
		gasOracle:     o.GasOracle,
		store:         o.Store,
		logg:          o.Logg,
		chainProvider: ethutils.NewProvider("https://offline.only", o.ChainID),
	}
}
