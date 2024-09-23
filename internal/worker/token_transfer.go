package worker

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	TokenTransferArgs struct {
		TrackingId   string `json:"trackingId"`
		From         string `json:"from"`
		To           string `json:"to"`
		TokenAddress string `json:"tokenAddress"`
		Amount       string `json:"amount"`
	}

	TokenTransferWorker struct {
		river.WorkerDefaults[TokenTransferArgs]
		store  store.Store
		logg   *slog.Logger
		signer *signer
	}
)

func (TokenTransferArgs) Kind() string { return store.TOKEN_TRANSFER }

func (w *TokenTransferWorker) Work(ctx context.Context, job *river.Job[TokenTransferArgs]) error {
	tx, err := w.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := w.store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	amount, err := StringToBigInt(job.Args.Amount)
	if err != nil {
		return err
	}

	input, err := abi[Transfer].EncodeArgs(
		ethutils.HexToAddress(job.Args.To),
		amount,
	)
	if err != nil {
		return err
	}

	gasSettings, err := w.signer.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.signer.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.TokenAddress),
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

	if err := w.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingId,
		OTXType:       store.TOKEN_TRANSFER,
		SignerAccount: job.Args.From,
		RawTx:         hexutil.Encode(rawTx),
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
