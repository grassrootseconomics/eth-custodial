package worker

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	ensclient "github.com/grassrootseconomics/eth-custodial/internal/ens_client"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/grassrootseconomics/ge-publish/pkg/contract"
	"github.com/lmittmann/w3"
	"github.com/riverqueue/river"
)

type (
	PoolDeployArgs struct {
		TrackingID string `json:"trackingId"`
		Name       string `json:"name"`
		Symbol     string `json:"symbol"`
		Owner      string `json:"owner"`
	}

	PoolDeployWorker struct {
		river.WorkerDefaults[PoolDeployArgs]
		wc *WorkerContainer
	}
)

func (PoolDeployArgs) Kind() string { return store.POOL_DEPLOY }

func (w *PoolDeployWorker) Work(ctx context.Context, job *river.Job[PoolDeployArgs]) error {
	tx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	systemKeypair, err := w.wc.store.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(systemKeypair.Private)
	if err != nil {
		return err
	}

	gasSettings, err := w.wc.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	// Deploy TokenIndex
	tokenIndexNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	tokenIndexContract := contract.NewTokenIndex()
	tokenIndexByteCode, err := tokenIndexContract.Bytecode()
	if err != nil {
		return err
	}

	builtTokenIndexDeployTx, err := w.wc.chainProvider.SignContractPublishTx(privateKey, ethutils.ContractPublishTxOpts{
		ContractByteCode: tokenIndexByteCode,
		GasFeeCap:        gasSettings.GasFeeCap,
		GasTipCap:        gasSettings.GasTipCap,
		GasLimit:         tokenIndexContract.MaxGasLimit(),
		Nonce:            tokenIndexNonce,
	})
	if err != nil {
		return err
	}

	rawTokenIndexDeployTx, err := builtTokenIndexDeployTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTokenIndexDeployTxHex := hexutil.Encode(rawTokenIndexDeployTx)
	tokenIndexAddress := crypto.CreateAddress(common.HexToAddress(systemKeypair.Public), tokenIndexNonce)

	tokenIndexDeployOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_INDEX_DEPLOY,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTokenIndexDeployTxHex,
		TxHash:        builtTokenIndexDeployTx.Hash().Hex(),
		Nonce:         tokenIndexNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  tokenIndexDeployOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Deploy Limiter
	limiterNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	limiterContract := contract.NewLimiter()
	limiterByteCode, err := limiterContract.Bytecode()
	if err != nil {
		return err
	}

	builtLimiterDeployTx, err := w.wc.chainProvider.SignContractPublishTx(privateKey, ethutils.ContractPublishTxOpts{
		ContractByteCode: limiterByteCode,
		GasFeeCap:        gasSettings.GasFeeCap,
		GasTipCap:        gasSettings.GasTipCap,
		GasLimit:         limiterContract.MaxGasLimit(),
		Nonce:            limiterNonce,
	})
	if err != nil {
		return err
	}

	rawLimiterDeployTx, err := builtLimiterDeployTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawLimiterDeployTxHex := hexutil.Encode(rawLimiterDeployTx)
	limiterAddress := crypto.CreateAddress(common.HexToAddress(systemKeypair.Public), limiterNonce)

	limiterDeployOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.LIMITER_DEPLOY,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawLimiterDeployTxHex,
		TxHash:        builtLimiterDeployTx.Hash().Hex(),
		Nonce:         limiterNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  limiterDeployOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Deploy SwapPool
	swapPoolNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	swapPoolContract := contract.NewSwapPool(contract.SwapPoolConstructorArgs{
		Name:          job.Args.Name,
		Symbol:        job.Args.Symbol,
		Decimals:      6,
		TokenRegistry: tokenIndexAddress,
		TokenLimiter:  limiterAddress,
	})
	swapPoolByteCode, err := swapPoolContract.Bytecode()
	if err != nil {
		return err
	}

	builtSwapPoolDeployTx, err := w.wc.chainProvider.SignContractPublishTx(privateKey, ethutils.ContractPublishTxOpts{
		ContractByteCode: swapPoolByteCode,
		GasFeeCap:        gasSettings.GasFeeCap,
		GasTipCap:        gasSettings.GasTipCap,
		GasLimit:         swapPoolContract.MaxGasLimit(),
		Nonce:            swapPoolNonce,
	})
	if err != nil {
		return err
	}

	rawSwapPoolDeployTx, err := builtSwapPoolDeployTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawSwapPoolDeployTxHex := hexutil.Encode(rawSwapPoolDeployTx)
	swapPoolAddress := crypto.CreateAddress(common.HexToAddress(systemKeypair.Public), swapPoolNonce)

	swapPoolDeployOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.SWAPPOOL_DEPLOY,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawSwapPoolDeployTxHex,
		TxHash:        builtSwapPoolDeployTx.Hash().Hex(),
		Nonce:         swapPoolNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  swapPoolDeployOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Deploy PriceIndexQuoter
	priceIndexQuoterNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	priceIndexQuoterContract := contract.NewPriceIndexQuoter()
	priceIndexQuoterByteCode, err := priceIndexQuoterContract.Bytecode()
	if err != nil {
		return err
	}

	builtPriceIndexQuoterDeployTx, err := w.wc.chainProvider.SignContractPublishTx(privateKey, ethutils.ContractPublishTxOpts{
		ContractByteCode: priceIndexQuoterByteCode,
		GasFeeCap:        gasSettings.GasFeeCap,
		GasTipCap:        gasSettings.GasTipCap,
		GasLimit:         priceIndexQuoterContract.MaxGasLimit(),
		Nonce:            priceIndexQuoterNonce,
	})
	if err != nil {
		return err
	}

	rawPriceIndexQuoterDeployTx, err := builtPriceIndexQuoterDeployTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawPriceIndexQuoterDeployTxHex := hexutil.Encode(rawPriceIndexQuoterDeployTx)
	priceIndexQuoterAddress := crypto.CreateAddress(common.HexToAddress(systemKeypair.Public), priceIndexQuoterNonce)

	priceIndexQuoterDeployOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.PRICEINDEXQUOTER_DEPLOY,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawPriceIndexQuoterDeployTxHex,
		TxHash:        builtPriceIndexQuoterDeployTx.Hash().Hex(),
		Nonce:         priceIndexQuoterNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  priceIndexQuoterDeployOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Add SwapPool to PoolIndex
	addToPoolIndexData, err := Abi[Add].EncodeArgs(swapPoolAddress)
	if err != nil {
		return err
	}

	addToPoolIndexNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	poolIndex := w.wc.registry[ethutils.PoolIndex]
	if w.wc.prod {
		poolIndex = w3.A("0x01eD8Fe01a2Ca44Cb26D00b1309d7D777471D00C")
	}

	builtAddToPoolIndexTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: poolIndex,
		InputData:       addToPoolIndexData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           addToPoolIndexNonce,
	})
	if err != nil {
		return err
	}

	rawAddToPoolIndexTx, err := builtAddToPoolIndexTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawAddToPoolIndexTxHex := hexutil.Encode(rawAddToPoolIndexTx)

	addToPoolIndexOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.POOL_INDEX_ADD,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawAddToPoolIndexTxHex,
		TxHash:        builtAddToPoolIndexTx.Hash().Hex(),
		Nonce:         addToPoolIndexNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  addToPoolIndexOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Call setQuoter on SwapPool
	setQuoterData, err := Abi[SetQuoter].EncodeArgs(priceIndexQuoterAddress)
	if err != nil {
		return err
	}

	setQuoterNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtSetQuoterTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: swapPoolAddress,
		InputData:       setQuoterData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           setQuoterNonce,
	})
	if err != nil {
		return err
	}

	rawSetQuoterTx, err := builtSetQuoterTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawSetQuoterTxHex := hexutil.Encode(rawSetQuoterTx)

	setQuoterOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.SET_QUOTER,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawSetQuoterTxHex,
		TxHash:        builtSetQuoterTx.Hash().Hex(),
		Nonce:         setQuoterNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  setQuoterOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	ownerAddress := common.HexToAddress(job.Args.Owner)

	// Transfer ownership of Limiter
	transferLimiterOwnershipData, err := Abi[TransferOwnership].EncodeArgs(ownerAddress)
	if err != nil {
		return err
	}

	transferLimiterOwnershipNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtTransferLimiterOwnershipTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: limiterAddress,
		InputData:       transferLimiterOwnershipData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           transferLimiterOwnershipNonce,
	})
	if err != nil {
		return err
	}

	rawTransferLimiterOwnershipTx, err := builtTransferLimiterOwnershipTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTransferLimiterOwnershipTxHex := hexutil.Encode(rawTransferLimiterOwnershipTx)

	transferLimiterOwnershipOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TRANSFER_OWNERSHIP,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTransferLimiterOwnershipTxHex,
		TxHash:        builtTransferLimiterOwnershipTx.Hash().Hex(),
		Nonce:         transferLimiterOwnershipNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  transferLimiterOwnershipOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Transfer ownership of TokenIndex
	transferTokenIndexOwnershipData, err := Abi[TransferOwnership].EncodeArgs(ownerAddress)
	if err != nil {
		return err
	}

	transferTokenIndexOwnershipNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtTransferTokenIndexOwnershipTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: tokenIndexAddress,
		InputData:       transferTokenIndexOwnershipData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           transferTokenIndexOwnershipNonce,
	})
	if err != nil {
		return err
	}

	rawTransferTokenIndexOwnershipTx, err := builtTransferTokenIndexOwnershipTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTransferTokenIndexOwnershipTxHex := hexutil.Encode(rawTransferTokenIndexOwnershipTx)

	transferTokenIndexOwnershipOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TRANSFER_OWNERSHIP,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTransferTokenIndexOwnershipTxHex,
		TxHash:        builtTransferTokenIndexOwnershipTx.Hash().Hex(),
		Nonce:         transferTokenIndexOwnershipNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  transferTokenIndexOwnershipOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Transfer ownership of SwapPool
	transferSwapPoolOwnershipData, err := Abi[TransferOwnership].EncodeArgs(ownerAddress)
	if err != nil {
		return err
	}

	transferSwapPoolOwnershipNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtTransferSwapPoolOwnershipTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: swapPoolAddress,
		InputData:       transferSwapPoolOwnershipData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           transferSwapPoolOwnershipNonce,
	})
	if err != nil {
		return err
	}

	rawTransferSwapPoolOwnershipTx, err := builtTransferSwapPoolOwnershipTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTransferSwapPoolOwnershipTxHex := hexutil.Encode(rawTransferSwapPoolOwnershipTx)

	transferSwapPoolOwnershipOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TRANSFER_OWNERSHIP,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTransferSwapPoolOwnershipTxHex,
		TxHash:        builtTransferSwapPoolOwnershipTx.Hash().Hex(),
		Nonce:         transferSwapPoolOwnershipNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  transferSwapPoolOwnershipOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	// Transfer ownership of PriceIndexQuoter
	transferPriceIndexQuoterOwnershipData, err := Abi[TransferOwnership].EncodeArgs(ownerAddress)
	if err != nil {
		return err
	}

	transferPriceIndexQuoterOwnershipNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtTransferPriceIndexQuoterOwnershipTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: priceIndexQuoterAddress,
		InputData:       transferPriceIndexQuoterOwnershipData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           transferPriceIndexQuoterOwnershipNonce,
	})
	if err != nil {
		return err
	}

	rawTransferPriceIndexQuoterOwnershipTx, err := builtTransferPriceIndexQuoterOwnershipTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTransferPriceIndexQuoterOwnershipTxHex := hexutil.Encode(rawTransferPriceIndexQuoterOwnershipTx)

	transferPriceIndexQuoterOwnershipOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TRANSFER_OWNERSHIP,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTransferPriceIndexQuoterOwnershipTxHex,
		TxHash:        builtTransferPriceIndexQuoterOwnershipTx.Hash().Hex(),
		Nonce:         transferPriceIndexQuoterOwnershipNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  transferPriceIndexQuoterOwnershipOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = w.wc.queueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      tokenIndexDeployOTXID,
				RawTx:      rawTokenIndexDeployTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      limiterDeployOTXID,
				RawTx:      rawLimiterDeployTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      swapPoolDeployOTXID,
				RawTx:      rawSwapPoolDeployTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      priceIndexQuoterDeployOTXID,
				RawTx:      rawPriceIndexQuoterDeployTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      addToPoolIndexOTXID,
				RawTx:      rawAddToPoolIndexTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 2,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      setQuoterOTXID,
				RawTx:      rawSetQuoterTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 3,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      transferLimiterOwnershipOTXID,
				RawTx:      rawTransferLimiterOwnershipTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 4,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      transferTokenIndexOwnershipOTXID,
				RawTx:      rawTransferTokenIndexOwnershipTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 4,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      transferSwapPoolOwnershipOTXID,
				RawTx:      rawTransferSwapPoolOwnershipTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 4,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      transferPriceIndexQuoterOwnershipOTXID,
				RawTx:      rawTransferPriceIndexQuoterOwnershipTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 4,
			},
		},
	})
	if err != nil {
		return err
	}

	// Best effort ENS registration
	ensHint := strings.ToLower(job.Args.Symbol) + ensclient.SuffixENSName
	ensInput := ensclient.RegisterInput{
		Address: swapPoolAddress.Hex(),
		Hint:    ensHint,
	}

	_, err = w.wc.ensClient.Register(ctx, ensInput)
	if err != nil {
		w.wc.logg.Warn("failed to register ENS name", "error", err, "hint", ensHint, "address", swapPoolAddress.Hex())
		// Don't fail the entire transaction if ENS registration fails
	} else {
		w.wc.logg.Info("successfully registered ENS name", "hint", ensHint, "address", swapPoolAddress.Hex())
	}

	w.wc.pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

	return tx.Commit(ctx)
}
