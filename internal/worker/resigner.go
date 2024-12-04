package worker

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/grassrootseconomics/ethutils"
)

func decodeTx(signedTxHash string) (*types.Transaction, error) {
	signedTxBytes, err := hexutil.Decode(signedTxHash)
	if err != nil {
		return nil, err
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(signedTxBytes); err != nil {
		return nil, err
	}

	return tx, nil
}

func bumpGas(signedTxHash string) (ethutils.ContractExecutionTxOpts, error) {
	originalTx, err := decodeTx(signedTxHash)
	if err != nil {
		return ethutils.ContractExecutionTxOpts{}, err
	}

	// In eth-custodial, we always sign with type 2 (dynamic fee) transactions
	if originalTx.Type() != types.DynamicFeeTxType {
		return ethutils.ContractExecutionTxOpts{}, errors.New("tx is not a dynamic fee transaction")
	}

	// Bump the gas price by 15%
	originalGasPrice := originalTx.GasFeeCap()
	bumpMultiplier := big.NewInt(115)
	bumped := new(big.Int).Mul(originalGasPrice, bumpMultiplier)
	bumped.Div(bumped, big.NewInt(100))

	if originalTx.To() == nil {
		return ethutils.ContractExecutionTxOpts{}, errors.New("tx has no recipient (is contract creation)")
	}

	return ethutils.ContractExecutionTxOpts{
		ContractAddress: *originalTx.To(),
		InputData:       originalTx.Data(),
		GasFeeCap:       bumped,
		GasTipCap:       originalTx.GasTipCap(),
		GasLimit:        originalTx.Gas(),
		Nonce:           originalTx.Nonce(),
	}, nil
}

func noopTx(gasFeeCap *big.Int, gasTipCap *big.Int, nonce uint64) ethutils.GasTransferTxOpts {
	return ethutils.GasTransferTxOpts{
		To:        ethutils.ZeroAddress,
		Value:     big.NewInt(0),
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
	}
}
