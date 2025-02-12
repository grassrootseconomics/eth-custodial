package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/ethutils"
	"github.com/kamikazechaser/jrpc"
	"github.com/riverqueue/river"
)

// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_sendtransaction
// DATA, 32 Bytes - the transaction hash, or the zero hash if the transaction is not yet available.
// We ignore other parameters for now as the custodial system takes care of them internally
type sendTransactionParams struct {
	To    string `json:"to"`
	From  string `json:"from"`
	Value string `json:"value"`
	Data  string `json:"data"`
}

func (a *API) methodEthSendTransaction(c jrpc.Context) error {
	var params []sendTransactionParams

	if err := c.Bind(&params); err != nil {
		return err
	}

	if len(params) < 1 {
		return jrpc.NewErrorInvalidParams("Atleast 1 param is required")
	}

	ctx := c.EchoContext().Request().Context()

	tx, err := a.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	keypair, err := a.store.LoadPrivateKey(ctx, tx, params[0].From)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	nonce, err := a.store.AcquireNonce(ctx, tx, params[0].From)
	if err != nil {
		return err
	}

	n, ok := new(big.Int).SetString(params[0].Value, 10)
	if !ok {
		return jrpc.NewErrorInvalidParams("Invalid value")
	}

	to := ethutils.HexToAddress(params[0].To)

	gasSettings, err := a.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	builtTx, err := types.SignNewTx(privateKey, a.chainProvider.Signer, &types.DynamicFeeTx{
		Value:     n,
		To:        &to,
		Nonce:     nonce,
		Data:      common.FromHex(params[0].Data),
		Gas:       gasSettings.GasLimit,
		GasFeeCap: gasSettings.GasFeeCap,
		GasTipCap: gasSettings.GasTipCap,
	})
	if err != nil {
		return err
	}

	rawTx, err := builtTx.MarshalBinary()
	if err != nil {
		return err
	}
	rawTxHex := hexutil.Encode(rawTx)

	trackindID := uuid.NewString()
	otxID, err := a.store.InsertOTX(ctx, tx, store.OTX{
		TrackingID:    trackindID,
		OTXType:       store.GENERIC_SIGN,
		SignerAccount: params[0].From,
		RawTx:         rawTxHex,
		TxHash:        builtTx.Hash().Hex(),
		Nonce:         nonce,
	})
	if err != nil {
		return err
	}

	if err := a.store.InsertDispatchTx(ctx, tx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.PENDING,
	}); err != nil {
		return err
	}

	_, err = a.queueClient.InsertManyTx(ctx, tx, []river.InsertManyParams{
		{
			Args: worker.DispatchArgs{
				TrackingID: trackindID,
				OTXID:      otxID,
				RawTx:      rawTxHex,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 1,
			},
		},
		{
			Args: worker.GasRefillArgs{
				TrackingID: trackindID,
				Address:    params[0].From,
			},
			InsertOpts: &river.InsertOpts{
				Priority: 2,
			},
		},
	})
	if err != nil {
		return handlePostgresError(c.EchoContext(), err)
	}

	if err := tx.Commit(ctx); err != nil {
		return handlePostgresError(c.EchoContext(), err)
	}

	a.logg.Debug("eth_sendTransaction request", "body", params, "tracking_id", trackindID)
	return c.Result(builtTx.Hash().Hex())
}
