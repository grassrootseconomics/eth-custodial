package gas

import (
	"math/big"
)

type (
	GasOpts struct {
		OracleType string
	}

	GasSettings struct {
		GasFeeCap *big.Int
		GasTipCap *big.Int
		GasLimit  uint64
	}

	GasOracle interface {
		GetSettings() (*GasSettings, error)
	}
)

func New(o GasOpts) GasOracle {
	var gasOracle GasOracle

	switch o.OracleType {
	case "static":
		gasOracle = &StaticGas{}
	default:
		gasOracle = &StaticGas{}
	}

	return gasOracle
}
