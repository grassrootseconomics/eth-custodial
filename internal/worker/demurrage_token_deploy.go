package worker

import (
	"context"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/grassrootseconomics/ethutils"
	"github.com/grassrootseconomics/ge-publish/pkg/contract"
	gepubutil "github.com/grassrootseconomics/ge-publish/pkg/util"
	"github.com/riverqueue/river"
)

type DemurrageTokenDeployArgs struct {
	TrackingID      string `json:"trackingId"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        uint8  `json:"decimals"`
	InitialSupply   string `json:"initialSupply"`
	InitialMintee   string `json:"initialMintee"`
	Owner           string `json:"owner"`
	SinkAddress     string `json:"sinkAddress"`
	DemurrageRate   string `json:"demurrageRate"`
	DemurragePeriod string `json:"demurragePeriod"`
}

type DemurrageTokenDeployWorker struct {
	river.WorkerDefaults[DemurrageTokenDeployArgs]
	wc         *WorkerContainer
	tokenIndex common.Address
}

func (DemurrageTokenDeployArgs) Kind() string { return store.DEMURRAGE_TOKEN_DEPLOY }

func (w *DemurrageTokenDeployWorker) Work(ctx context.Context, job *river.Job[DemurrageTokenDeployArgs]) error {
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

	nonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	demurrageRate, err := strconv.ParseInt(job.Args.DemurrageRate, 10, 64)
	if err != nil {
		return err
	}
	demurragePeriod, err := strconv.ParseInt(job.Args.DemurragePeriod, 10, 64)
	if err != nil {
		return err
	}
	decayLevel, err := gepubutil.CalculateDecayLevel(demurrageRate, demurragePeriod)
	if err != nil {
		return err
	}

	contract := contract.NewERC20Demurrage(contract.ERC20DemurrageConstructorArgs{
		Name:               job.Args.Name,
		Symbol:             job.Args.Symbol,
		Decimals:           job.Args.Decimals,
		DecayLevel:         decayLevel,
		PeriodMinutes:      big.NewInt(demurragePeriod),
		DefaultSinkAddress: common.HexToAddress(job.Args.SinkAddress),
	})
	byteCode, err := contract.Bytecode()
	if err != nil {
	}

	builtContractDeployTx, err := w.wc.chainProvider.SignContractPublishTx(privateKey, ethutils.ContractPublishTxOpts{
		ContractByteCode: byteCode,
		GasFeeCap:        gasSettings.GasFeeCap,
		GasTipCap:        gasSettings.GasTipCap,
		GasLimit:         contract.MaxGasLimit(),
		Nonce:            nonce,
	})
	if err != nil {
		return err
	}

	rawContractDeployTx, err := builtContractDeployTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawContractDeployTxHex := hexutil.Encode(rawContractDeployTx)

	deployContractOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.DEMURRAGE_TOKEN_DEPLOY,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawContractDeployTxHex,
		TxHash:        builtContractDeployTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  deployContractOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}
	w.wc.pub.Send(ctx, event.Event{
		TrackingID: job.Args.TrackingID,
		Status:     store.PENDING,
	})

	contractAddress := crypto.CreateAddress(common.HexToAddress(systemKeypair.Public), nonce)

	addData, err := Abi[Add].EncodeArgs(contractAddress)
	if err != nil {
		return err
	}

	addNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtAddTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: w.tokenIndex,
		InputData:       addData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           addNonce,
	})
	if err != nil {
		return err
	}

	rawAddTx, err := builtAddTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawAddTxHex := hexutil.Encode(rawAddTx)

	addOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_INDEX_ADD,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawAddTxHex,
		TxHash:        builtAddTx.Hash().Hex(),
		Nonce:         addNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  addOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	initialSupply, err := StringToBigInt(job.Args.InitialSupply, false)
	if err != nil {
		return err
	}

	mintToData, err := Abi[MintTo].EncodeArgs(
		common.HexToAddress(job.Args.InitialMintee),
		initialSupply,
	)
	if err != nil {
		return err
	}

	mintToNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtMintToTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: contractAddress,
		InputData:       mintToData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           mintToNonce,
	})
	if err != nil {
		return err
	}

	rawMintToTx, err := builtMintToTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawMintToTxHex := hexutil.Encode(rawMintToTx)

	mintToOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TOKEN_TRANSFER,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawMintToTxHex,
		TxHash:        builtMintToTx.Hash().Hex(),
		Nonce:         mintToNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  mintToOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	transferOwnershipData, err := Abi[TransferOwnership].EncodeArgs(
		common.HexToAddress(job.Args.Owner),
	)
	if err != nil {
		return err
	}

	transferOwnershipNonce, err := w.wc.store.AcquireNonce(ctx, tx, systemKeypair.Public)
	if err != nil {
		return err
	}

	builtTransferOwnershipTx, err := w.wc.chainProvider.SignContractExecutionTx(privateKey, ethutils.ContractExecutionTxOpts{
		ContractAddress: contractAddress,
		InputData:       transferOwnershipData,
		GasFeeCap:       gasSettings.GasFeeCap,
		GasTipCap:       gasSettings.GasTipCap,
		GasLimit:        gasSettings.GasLimit,
		Nonce:           transferOwnershipNonce,
	})
	if err != nil {
		return err
	}

	rawTransferOwnershipTx, err := builtTransferOwnershipTx.MarshalBinary()
	if err != nil {
		return err
	}

	rawTransferOwnershipTxHex := hexutil.Encode(rawTransferOwnershipTx)

	transferOwnershipOTXID, err := w.wc.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    job.Args.TrackingID,
		OTXType:       store.TRANSFER_OWNERSHIP,
		SignerAccount: systemKeypair.Public,
		RawTx:         rawTransferOwnershipTxHex,
		TxHash:        builtTransferOwnershipTx.Hash().Hex(),
		Nonce:         transferOwnershipNonce,
	})
	if err != nil {
		return err
	}

	if err := w.wc.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  transferOwnershipOTXID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = w.wc.queueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      deployContractOTXID,
				RawTx:      rawContractDeployTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      addOTXID,
				RawTx:      rawAddTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 2,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      mintToOTXID,
				RawTx:      rawMintToTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 3,
			},
		},
		{
			Args: DispatchArgs{
				TrackingID: job.Args.TrackingID,
				OTXID:      transferOwnershipOTXID,
				RawTx:      rawTransferOwnershipTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 4,
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
