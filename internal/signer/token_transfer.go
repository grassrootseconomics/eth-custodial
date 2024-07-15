package signer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/grassrootseconomics/ethutils"
	"github.com/jackc/pgx/v5"
)

func (s *Signer) SignTokenTransfer(ctx context.Context, pgxTx pgx.Tx, payload TokenTransferPayload) (*types.Transaction, error) {
	key, err := s.store.LoadPrivateKey(ctx, payload.From)
	if err != nil {
		return nil, err
	}

	nonce, err := s.store.AcquireNonce(ctx, payload.From)
	if err != nil {
		return nil, err
	}

	input, err := abi[Transfer].EncodeArgs(
		ethutils.HexToAddress(payload.To),
		new(big.Int).SetUint64(payload.Amount),
	)
	if err != nil {
		return nil, err
	}

	gasSettings, err := s.gasOracle.GetSettings()
	if err != nil {
		return nil, err
	}

	builtTx, err := s.provider.SignContractExecutionTx(key, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(payload.VoucherAddress),
		InputData:       input,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           nonce,
	})
	if err != nil {
		return nil, err
	}

	return builtTx, nil
}
