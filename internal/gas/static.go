package gas

import "github.com/grassrootseconomics/ethutils"

type StaticGas struct{}

func (sg *StaticGas) GetSettings() (*GasSettings, error) {
	return &GasSettings{
		GasFeeCap: ethutils.SafeGasFeeCap,
		GasTipCap: ethutils.SafeGasTipCap,
		GasLimit:  uint64(ethutils.SafeGasLimit),
	}, nil
}
