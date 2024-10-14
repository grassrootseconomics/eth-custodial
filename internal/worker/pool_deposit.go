package worker

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/ethutils"
	"github.com/riverqueue/river"
)

// 0xE1CB2DbaB6b6feFe34B2AaA47b83EeE79E8D36f8 - cUSD
// 0x36C77CC277e73CCcd199d1989828739722Fe5450 - testToken
// 0x6E757C04Dcf764f7b06AFC428eD75773CDAC7fDc  - pool
// 0x47690CC4C0c91be612A643D85195F4D57488dE97-  limiter
// 0x0bca6353f80bBe401233d5Ed7732cEACEb13bc88 - quoter
type (
	PoolDepositArgs struct {
		TrackingID   string `json:"trackingId"`
		From         string `json:"from"`
		TokenAddress string `json:"tokenAddress"`
		PoolAddress  string `json:"poolAddress"`
		Amount       string `json:"amount"`
	}

	PoolDepositWorker struct {
		river.WorkerDefaults[PoolDepositArgs]
		wc *WorkerContainer
	}
)

func (PoolDepositArgs) Kind() string { return store.POOL_DEPSOIT }

func (w *PoolDepositWorker) Work(ctx context.Context, job *river.Job[PoolDepositArgs]) error {
	tx, err := w.wc.Store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := w.wc.Store.LoadPrivateKey(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	gasSettings, err := w.wc.GasOracle.GetSettings()
	if err != nil {
		return err
	}

	// Reset approval -> 0

	resetApprovalNonce, err := w.wc.Store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	resetApprovalInput, err := abi[Approve].EncodeArgs(
		ethutils.HexToAddress(job.Args.PoolAddress),
		big.NewInt(0),
	)
	if err != nil {
		return err
	}

	builtResetApprovalTx, err := w.wc.ChainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.TokenAddress),
		InputData:       resetApprovalInput,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           resetApprovalNonce,
	})
	if err != nil {
		return err
	}

	rawResetApprovalTx, err := builtResetApprovalTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawResetApprovalTxHex := hexutil.Encode(rawResetApprovalTx)

	resetApprovalOTXID, err := w.wc.Store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_APPROVE,
		SignerAccount: job.Args.From,
		RawTx:         rawResetApprovalTxHex,
		TxHash:        builtResetApprovalTx.Hash().Hex(),
		Nonce:         resetApprovalNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.Store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  resetApprovalOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Set approval -> amount + 5%

	setApprovalNonce, err := w.wc.Store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	bumpedApprovalAmount, err := StringToBigInt(job.Args.Amount, true)
	if err != nil {
		return err
	}

	setApprovalInput, err := abi[Approve].EncodeArgs(
		ethutils.HexToAddress(job.Args.PoolAddress),
		bumpedApprovalAmount,
	)
	if err != nil {
		return err
	}

	builtSetApprovalTx, err := w.wc.ChainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.TokenAddress),
		InputData:       setApprovalInput,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           setApprovalNonce,
	})
	if err != nil {
		return err
	}

	rawSetApprovalTx, err := builtSetApprovalTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawSetApprovalTxHex := hexutil.Encode(rawSetApprovalTx)

	setApprovalOTXID, err := w.wc.Store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_APPROVE,
		SignerAccount: job.Args.From,
		RawTx:         rawSetApprovalTxHex,
		TxHash:        builtSetApprovalTx.Hash().Hex(),
		Nonce:         setApprovalNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.Store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  setApprovalOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Initiate swap

	nonce, err := w.wc.Store.AcquireNonce(ctx, tx, job.Args.From)
	if err != nil {
		return err
	}

	amount, err := StringToBigInt(job.Args.Amount, false)
	if err != nil {
		return err
	}

	input, err := abi[Deposit].EncodeArgs(
		ethutils.HexToAddress(job.Args.TokenAddress),
		amount,
	)
	if err != nil {
		return err
	}

	builtTx, err := w.wc.ChainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: ethutils.HexToAddress(job.Args.PoolAddress),
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

	otxID, err := w.wc.Store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.POOL_DEPSOIT,
		SignerAccount: job.Args.From,
		RawTx:         rawTxHex,
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.Store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}
	_, err = w.wc.QueueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{Args: DispatchArgs{
			TrackingID: job.Args.TrackingID,
			OTXID:      resetApprovalOTXID,
			RawTx:      rawResetApprovalTxHex,
		}, InsertOpts: &river.InsertOpts{Priority: 1}},
		{Args: DispatchArgs{
			TrackingID: job.Args.TrackingID,
			OTXID:      setApprovalOTXID,
			RawTx:      rawSetApprovalTxHex,
		}, InsertOpts: &river.InsertOpts{Priority: 2}},
		{Args: DispatchArgs{
			TrackingID: job.Args.TrackingID,
			OTXID:      otxID,
			RawTx:      rawTxHex,
		}, InsertOpts: &river.InsertOpts{Priority: 3}},
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}