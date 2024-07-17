package worker

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/grassrootseconomics/ethutils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type (
	TokenTransferArgs struct {
		TrackingId     string `json:"trackingId"`
		From           string `json:"from"`
		To             string `json:"to"`
		VoucherAddress string `json:"tokenAddress"`
		Amount         uint64 `json:"amount"`
	}

	TokenTransferWorker struct {
		river.WorkerDefaults[TokenTransferArgs]
		pgxPool *pgxpool.Pool
		signer  *signer
	}
)

const tokenTransferID = "TOKEN_TRANSFER"

func (TokenTransferArgs) Kind() string { return tokenTransferID }

func (w *TokenTransferWorker) Work(ctx context.Context, job *river.Job[TokenTransferArgs]) error {
	tx, err := w.pgxPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	key, err := w.signer.store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	nonce, err := w.signer.store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	input, err := abi[Transfer].EncodeArgs(
		ethutils.HexToAddress(job.Args.To),
		new(big.Int).SetUint64(job.Args.Amount),
	)
	if err != nil {
		return err
	}

	gasSettings, err := w.signer.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.signer.chainProvider.SignContractExecutionTx(key, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.VoucherAddress),
		InputData:       input,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           nonce,
	})
	if err != nil {
		return err
	}

	rawTx, err := builtTx.MarshalBinary()
	if err != nil {
		return err
	}

	if err := w.signer.store.InsertOTX(ctx, tx, builtTx.Hash().Hex(), hexutil.Encode(rawTx)); err != nil {
		return nil
	}

	return nil
}
