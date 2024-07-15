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

	Gas interface {
		GetSettings() (*GasSettings, error)
	}
)

func New(o GasOpts) Gas {
	var gasOracle Gas

	switch o.OracleType {
	case "static":
		gasOracle = &StaticGas{}
	default:
		gasOracle = &StaticGas{}
	}

	return gasOracle
}
