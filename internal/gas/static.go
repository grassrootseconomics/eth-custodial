package gas

import (
	"math/big"

	"github.com/grassrootseconomics/ethutils"
)

type StaticGas struct{}

func (sg *StaticGas) GetSettings() (*GasSettings, error) {
	return &GasSettings{
		GasFeeCap: big.NewInt(15000000000),
		GasTipCap: ethutils.SafeGasTipCap,
		GasLimit:  uint64(ethutils.SafeGasLimit),
	}, nil
}

func (sg *StaticGas) Start() {}
func (sg *StaticGas) Stop()  {}
