package worker

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/riverqueue/river"
)

type (
	GasRefillArgs struct {
		OTXID      uint64 `json:"otxId"`
		TrackingID string `json:"trackingId"`
		Address    string `json:"address"`
	}

	GasRefillWorker struct {
		river.WorkerDefaults[GasRefillArgs]
		wc        *WorkerContainer
		gasFaucet common.Address
	}
)

func (GasRefillArgs) Kind() string { return store.GAS_REFILL }

func (w *GasRefillWorker) Work(ctx context.Context, job *river.Job[GasRefillArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var (
		nextTime    *big.Int
		checkStatus bool
	)

	if err := w.wc.chainProvider.Client.CallCtx(
		ctx,
		eth.CallFunc(
			w.gasFaucet,
			Abi[NextTime],
			w3.A(job.Args.Address),
		).Returns(&nextTime),
	); err != nil {
		return err
	}

	if nextTime.Int64() > time.Now().Unix() {
		w.wc.logg.Info("gas refill not needed", "address", job.Args.Address)
		return nil
	}

	if err := w.wc.chainProvider.Client.CallCtx(
		ctx,
		eth.CallFunc(
			w.gasFaucet,
			Abi[Check],
			w3.A(job.Args.Address),
		).Returns(&checkStatus),
	); err != nil {
		return err
	}

	if !checkStatus {
		w.wc.logg.Warn("gas poke check fail", "address", job.Args.Address)
		return nil
	}

	systemKeypair, err := w.wc.store.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(systemKeypair.Private)
	if err != nil {
		return err
	}

	nonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	input, err := Abi[GiveTo].EncodeArgs(w3.A(job.Args.Address))
	if err != nil {
		return err
	}

	gasSettings, err := w.wc.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: w.gasFaucet,
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
		OTXType:       store.GAS_REFILL,
		SignerAccount: systemKeypair.Public,
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

	w.wc.pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

	_, err = w.wc.queueClient.InsertTx(ctx, tx, DispatchArgs{
		TrackingID: job.Args.TrackingID,
		OTXID:      otxID,
		RawTx:      rawTxHex,
	}, nil)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
