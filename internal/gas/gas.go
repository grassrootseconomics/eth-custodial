package gas

import (
	"math/big"
)

type (
	GasSettings struct {
		GasFeeCap *big.Int
		GasTipCap *big.Int
		GasLimit  uint64
	}

	GasOracle interface {
		GetSettings() (*GasSettings, error)
		Start()
		Stop()
	}
)
