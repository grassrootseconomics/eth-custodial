package worker

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

type (
	TokenTransferArgs struct {
		TrackingID   string `json:"trackingId"`
		From         string `json:"from"`
		To           string `json:"to"`
		TokenAddress string `json:"tokenAddress"`
		Amount       string `json:"amount"`
	}

	TokenTransferWorker struct {
		river.WorkerDefaults[TokenTransferArgs]
		wc *WorkerContainer
	}
)

func (TokenTransferArgs) Kind() string { return store.TOKEN_TRANSFER }

func (w *TokenTransferWorker) Work(ctx context.Context, job *river.Job[TokenTransferArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := w.wc.store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.wc.store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	amount, err := StringToBigInt(job.Args.Amount, false)
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

	gasSettings, err := w.wc.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
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

	rawTxHex := hexutil.Encode(rawTx)

	otxID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_TRANSFER,
		SignerAccount: job.Args.From,
		RawTx:         rawTxHex,
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = w.wc.queueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      otxID,
				RawTx:      rawTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: GasRefillArgs{
				TrackingID: job.Args.TrackingID,
				Address:    job.Args.From,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 2,
			},
		},
	})
	if err != nil {
		return err
	}

	w.wc.pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

	return tx.Commit(ctx)
}
