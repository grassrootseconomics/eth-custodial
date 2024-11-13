package gas

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/grassrootseconomics/ethutils"
	"github.com/lmittmann/w3/module/eth"
)

type (
	RPCGasOracleOpts struct {
		Logg          *slog.Logger
		ChainProvider *ethutils.Provider
	}

	RPCGasOracle struct {
		logg           *slog.Logger
		chainProvider  *ethutils.Provider
		cachedGasPrice *GasSettings
		stopCh         chan struct{}
	}
)

const rpcUpdateInterval = 30 * time.Second

func NewRPCGasOracle(o RPCGasOracleOpts) (*RPCGasOracle, error) {
	rpcGasOracle := &RPCGasOracle{
		logg:          o.Logg,
		chainProvider: o.ChainProvider,
		stopCh:        make(chan struct{}),
		cachedGasPrice: &GasSettings{
			GasLimit: uint64(ethutils.SafeGasLimit),
		},
	}

	if err := rpcGasOracle.updateGasPrice(); err != nil {
		return nil, err
	}

	return rpcGasOracle, nil
}

func (g *RPCGasOracle) Stop() {
	g.stopCh <- struct{}{}
}

func (g *RPCGasOracle) Start() {
	ticker := time.NewTicker(rpcUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			g.logg.Debug("stopping rpc gas oracle updater")
			return
		case <-ticker.C:
			if err := g.updateGasPrice(); err != nil {
				g.logg.Error("failed to update rpc gas price", "err", err)
			}

		}
	}
}

func (g *RPCGasOracle) GetSettings() (*GasSettings, error) {
	return g.cachedGasPrice, nil
}

func (g *RPCGasOracle) updateGasPrice() error {
	var (
		newGasPrice *big.Int
		newTipCap   *big.Int
	)

	if err := g.chainProvider.Client.CallCtx(
		context.Background(),
		eth.GasPrice().Returns(&newGasPrice),
		eth.GasTipCap().Returns(&newTipCap),
	); err != nil {
		return err
	}

	// We pay 20% more than the current gas price to accomdate any fluctuations between cache updates
	bumpFactor := big.NewInt(120)
	newGasPrice = newGasPrice.Mul(newGasPrice, bumpFactor)
	newGasPrice = newGasPrice.Div(newGasPrice, big.NewInt(100))

	g.cachedGasPrice.GasFeeCap = newGasPrice
	g.cachedGasPrice.GasTipCap = newTipCap
	g.logg.Debug("updated rpc gas price", "gas_fee_cap", g.cachedGasPrice.GasFeeCap, "gas_tip_cap", g.cachedGasPrice.GasTipCap)

	return nil
}
