package signer

import (
	"log/slog"

	"github.com/grassrootseconomics/celo-custodial/internal/gas"
	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
)

type (
	SignerOpts struct {
		ChainID   int64
		GasOracle gas.GasOracle
		Store     store.Store
		Logg      *slog.Logger
	}

	Signer struct {
		gasOracle gas.GasOracle
		store     store.Store
		logg      *slog.Logger
		provider  *ethutils.Provider
	}
)

func New(o SignerOpts) *Signer {
	return &Signer{
		logg:     o.Logg,
		provider: ethutils.NewProvider("https://offline.only", o.ChainID),
	}
}
