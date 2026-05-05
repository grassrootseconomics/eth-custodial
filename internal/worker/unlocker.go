package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/riverqueue/river"
)

type (
	UnlockerArgs struct{}

	UnlockerWorker struct {
		river.WorkerDefaults[UnlockerArgs]
		wc *WorkerContainer
	}
)

const UnlockerID = "UNLOCKER"

func (UnlockerArgs) Kind() string { return UnlockerID }

func (w *UnlockerWorker) Work(ctx context.Context, _ *river.Job[UnlockerArgs]) error {
	stuckOTXs, err := w.getStuckOTXs(ctx)
	if err != nil {
		return err
	}

	if len(stuckOTXs) == 0 {
		w.wc.logg.Debug("unlocker: no stuck transactions older than configured threshold", "threshold", unlockerInterval)
		return nil
	}

	accounts := affectedAccounts(stuckOTXs)
	w.wc.logg.Info("unlocker: processing affected accounts", "count", len(accounts))

	for account := range accounts {
		if err := w.processAccount(ctx, account); err != nil {
			w.wc.logg.Error("unlocker: failed to process account", "account", account, "error", err)
		}
	}

	return nil
}

func (w *UnlockerWorker) processAccount(ctx context.Context, account string) error {
	var networkNonce uint64
	if err := w.wc.chainProvider.Client.CallCtx(
		ctx,
		eth.Nonce(common.HexToAddress(account), nil).Returns(&networkNonce),
	); err != nil {
		return err
	}
	w.wc.logg.Info("unlocker: network nonce", "account", account, "nonce", networkNonce)

	otxs, err := w.getOTXsFromNonce(ctx, account, networkNonce)
	if err != nil {
		return err
	}

	if len(otxs) == 0 {
		return nil
	}
	w.wc.logg.Info("unlocker: resubmitting from nonce", "account", account, "count", len(otxs), "from_nonce", otxs[0].Nonce)

	for _, otx := range otxs {
		w.wc.logg.Info("unlocker: processing OTX",
			"otx_id", otx.ID,
			"nonce", otx.Nonce,
			"status", otx.DispatchStatus,
			"type", otx.OTXType,
		)

		if err := w.processOTX(ctx, otx); err != nil {
			w.wc.logg.Error("unlocker: failed to process OTX, stopping account sequence",
				"otx_id", otx.ID,
				"nonce", otx.Nonce,
				"error", err,
			)
			return nil
		}
	}

	return nil
}

func (w *UnlockerWorker) processOTX(ctx context.Context, otx *store.OTX) error {
	rawTxBytes, err := hexutil.Decode(otx.RawTx)
	if err != nil {
		return err
	}

	if err := w.sendRawTx(ctx, rawTxBytes); err == nil {
		w.wc.logg.Info("unlocker: resubmitted successfully", "otx_id", otx.ID, "nonce", otx.Nonce)
		return w.setStatus(ctx, otx.ID, store.IN_NETWORK)
	} else {
		errType := unlockerClassifyRPCError(err)
		w.wc.logg.Info("unlocker: resubmit error", "otx_id", otx.ID, "type", errType, "error", err)

		switch errType {
		case "nonce_low":
			return w.checkReceipt(ctx, otx)

		case "gas_price", "replacement_underpriced":
			return w.resignAndResubmit(ctx, otx)

		case "no_gas":
			w.wc.logg.Warn("unlocker: account has no gas, stopping sequence", "otx_id", otx.ID)
			return nil

		default:
			return err
		}
	}
}

func (w *UnlockerWorker) sendRawTx(ctx context.Context, rawTx []byte) error {
	var txHash common.Hash
	var callErrs w3.CallErrors

	if err := w.wc.chainProvider.Client.CallCtx(ctx, eth.SendRawTx(rawTx).Returns(&txHash)); errors.As(err, &callErrs) {
		if jsonErr, ok := callErrs[0].(rpc.Error); ok {
			if classified := handleJSONRPCError(jsonErr.Error()); classified != nil {
				return &DispatchError{Err: classified, OriginalErr: jsonErr}
			}
		}
		return callErrs[0]
	} else if err != nil {
		return handleNetworkError(err)
	}

	return nil
}

func (w *UnlockerWorker) checkReceipt(ctx context.Context, otx *store.OTX) error {
	var receipt *types.Receipt
	if err := w.wc.chainProvider.Client.CallCtx(
		ctx,
		eth.TxReceipt(common.HexToHash(otx.TxHash)).Returns(&receipt),
	); err != nil {
		return err
	}

	if receipt == nil {
		// Stored hash was never mined. Check if nonce was consumed by a different tx
		// (resignAndResubmit race: replacement submitted, DB updated, but original got mined first).
		var networkNonce uint64
		if err := w.wc.chainProvider.Client.CallCtx(
			ctx,
			eth.Nonce(common.HexToAddress(otx.SignerAccount), nil).Returns(&networkNonce),
		); err != nil {
			return err
		}
		if networkNonce > otx.Nonce {
			return w.recoverOrphanedNonce(ctx, otx)
		}
		return nil
	}

	status := store.REVERTED
	if receipt.Status == 1 {
		status = store.SUCCESS
	}
	return w.setStatus(ctx, otx.ID, status)
}

// recoverOrphanedNonce handles the case where the stored tx_hash was never mined but the nonce
// was consumed on-chain by a different tx (resignAndResubmit block-inclusion race).
// It finds the actual tx by anchoring to the next confirmed OTX's block, then updates the DB.
func (w *UnlockerWorker) recoverOrphanedNonce(ctx context.Context, otx *store.OTX) error {
	w.wc.logg.Warn("unlocker: orphaned nonce detected, attempting block scan recovery",
		"otx_id", otx.ID, "nonce", otx.Nonce, "account", otx.SignerAccount, "stored_hash", otx.TxHash)

	var anchorTxHash string
	if err := w.wc.store.Pool().QueryRow(ctx, `
		SELECT otx.tx_hash FROM otx
		INNER JOIN keystore ON otx.signer_account = keystore.id
		INNER JOIN dispatch ON otx.id = dispatch.otx_id
		WHERE keystore.public_key = $1 AND otx.nonce = $2 AND dispatch.status = 'SUCCESS'
		LIMIT 1`, otx.SignerAccount, otx.Nonce+1).Scan(&anchorTxHash); err != nil {
		w.wc.logg.Error("unlocker: orphaned nonce, no anchor tx found",
			"otx_id", otx.ID, "nonce", otx.Nonce)
		return nil
	}

	var anchorReceipt *types.Receipt
	if err := w.wc.chainProvider.Client.CallCtx(ctx,
		eth.TxReceipt(common.HexToHash(anchorTxHash)).Returns(&anchorReceipt)); err != nil || anchorReceipt == nil {
		w.wc.logg.Error("unlocker: orphaned nonce, anchor receipt unavailable",
			"otx_id", otx.ID, "nonce", otx.Nonce)
		return nil
	}

	anchorBlock := anchorReceipt.BlockNumber.Uint64()
	scanFrom := anchorBlock
	if anchorBlock > 5 {
		scanFrom = anchorBlock - 5
	}

	for blockNum := scanFrom; blockNum <= anchorBlock; blockNum++ {
		var block *types.Block
		if err := w.wc.chainProvider.Client.CallCtx(ctx,
			eth.BlockByNumber(new(big.Int).SetUint64(blockNum)).Returns(&block)); err != nil || block == nil {
			continue
		}
		for _, tx := range block.Transactions() {
			if tx.Nonce() != otx.Nonce {
				continue
			}
			sender, err := types.Sender(w.wc.chainProvider.Signer, tx)
			if err != nil || !strings.EqualFold(sender.Hex(), otx.SignerAccount) {
				continue
			}
			actualHash := tx.Hash().Hex()
			var actualReceipt *types.Receipt
			if err := w.wc.chainProvider.Client.CallCtx(ctx,
				eth.TxReceipt(tx.Hash()).Returns(&actualReceipt)); err != nil || actualReceipt == nil {
				continue
			}
			status := store.REVERTED
			if actualReceipt.Status == 1 {
				status = store.SUCCESS
			}
			w.wc.logg.Info("unlocker: orphaned nonce recovered",
				"otx_id", otx.ID, "nonce", otx.Nonce, "actual_hash", actualHash, "status", status)
			dbTx, err := w.wc.store.Pool().Begin(ctx)
			if err != nil {
				return err
			}
			defer dbTx.Rollback(ctx)
			if _, err := dbTx.Exec(ctx, `UPDATE otx SET tx_hash = $1 WHERE id = $2`, actualHash, otx.ID); err != nil {
				return err
			}
			if err := w.wc.store.UpdateDispatchTxStatus(ctx, dbTx, store.DispatchTx{
				OTXID:  otx.ID,
				Status: status,
			}); err != nil {
				return err
			}
			return dbTx.Commit(ctx)
		}
	}

	w.wc.logg.Error("unlocker: orphaned nonce, tx not found in block scan",
		"otx_id", otx.ID, "nonce", otx.Nonce, "scanned_blocks", fmt.Sprintf("%d-%d", scanFrom, anchorBlock))
	return nil
}

func (w *UnlockerWorker) resignAndResubmit(ctx context.Context, otx *store.OTX) error {
	originalTxBytes, err := hexutil.Decode(otx.RawTx)
	if err != nil {
		return err
	}

	originalTx := new(types.Transaction)
	if err := originalTx.UnmarshalBinary(originalTxBytes); err != nil {
		return err
	}

	if originalTx.To() == nil {
		return nil
	}

	if originalTx.Type() != types.DynamicFeeTxType {
		return errors.New("cannot re-sign non-dynamic-fee transaction")
	}

	gasSettings, err := w.wc.gasOracle.GetSettings()
	if err != nil {
		return err
	}

	newGasFeeCap := gasSettings.GasFeeCap
	newGasTipCap := gasSettings.GasTipCap

	if originalTx.GasFeeCap() != nil && newGasFeeCap.Cmp(originalTx.GasFeeCap()) <= 0 {
		bump := new(big.Int).Mul(originalTx.GasFeeCap(), big.NewInt(115))
		newGasFeeCap = bump.Div(bump, big.NewInt(100))
	}
	if originalTx.GasTipCap() != nil && newGasTipCap.Cmp(originalTx.GasTipCap()) <= 0 {
		bump := new(big.Int).Mul(originalTx.GasTipCap(), big.NewInt(115))
		newGasTipCap = bump.Div(bump, big.NewInt(100))
	}

	dbTx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	keypair, err := w.wc.store.LoadPrivateKey(ctx, dbTx, otx.SignerAccount)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return err
	}

	newTx, err := types.SignNewTx(privateKey, w.wc.chainProvider.Signer, &types.DynamicFeeTx{
		Nonce:     originalTx.Nonce(),
		To:        originalTx.To(),
		Value:     originalTx.Value(),
		Data:      originalTx.Data(),
		Gas:       originalTx.Gas(),
		GasFeeCap: newGasFeeCap,
		GasTipCap: newGasTipCap,
	})
	if err != nil {
		return err
	}

	newRawTxBytes, err := newTx.MarshalBinary()
	if err != nil {
		return err
	}

	if err := w.sendRawTx(ctx, newRawTxBytes); err != nil {
		return err
	}

	// Guard against the block-inclusion race: if the original tx was sealed into a block
	// while we were re-signing, our new tx is now invalid. Recover via checkReceipt.
	var chainNonce uint64
	if err := w.wc.chainProvider.Client.CallCtx(ctx,
		eth.Nonce(common.HexToAddress(otx.SignerAccount), nil).Returns(&chainNonce)); err == nil && chainNonce > otx.Nonce {
		w.wc.logg.Warn("unlocker: nonce consumed during re-sign, recovering from original",
			"otx_id", otx.ID, "nonce", otx.Nonce)
		return w.checkReceipt(ctx, otx)
	}

	newRawTxHex := hexutil.Encode(newRawTxBytes)
	newTxHash := newTx.Hash().Hex()

	if _, err := dbTx.Exec(ctx,
		`UPDATE otx SET raw_tx = $1, tx_hash = $2 WHERE id = $3`,
		newRawTxHex, newTxHash, otx.ID,
	); err != nil {
		return err
	}

	if err := w.wc.store.UpdateDispatchTxStatus(ctx, dbTx, store.DispatchTx{
		OTXID:  otx.ID,
		Status: store.IN_NETWORK,
	}); err != nil {
		return err
	}

	w.wc.logg.Info("unlocker: re-signed and resubmitted", "otx_id", otx.ID, "new_tx_hash", newTxHash)
	return dbTx.Commit(ctx)
}

func (w *UnlockerWorker) setStatus(ctx context.Context, otxID uint64, status string) error {
	dbTx, err := w.wc.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	if err := w.wc.store.UpdateDispatchTxStatus(ctx, dbTx, store.DispatchTx{
		OTXID:  otxID,
		Status: status,
	}); err != nil {
		return err
	}

	return dbTx.Commit(ctx)
}

func (w *UnlockerWorker) getStuckOTXs(ctx context.Context) ([]*store.OTX, error) {
	cutoff := time.Now().Add(-unlockerInterval)
	rows, err := w.wc.store.Pool().Query(ctx, `
		SELECT otx.id, otx.tracking_id, otx.otx_type, keystore.public_key, otx.raw_tx, otx.tx_hash,
		       otx.nonce, otx.replaced, otx.created_at, otx.updated_at, dispatch.status
		FROM keystore
		INNER JOIN otx ON keystore.id = otx.signer_account
		INNER JOIN dispatch ON otx.id = dispatch.otx_id
		WHERE dispatch.status NOT IN ('SUCCESS', 'REVERTED', 'EXTERNAL_DISPATCH')
		  AND otx.otx_type NOT IN ('GENERIC_SIGN', 'OTHER_MANUAL')
		  AND dispatch.updated_at <= $1
		ORDER BY otx.id ASC
		LIMIT 100`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOTXRows(rows)
}

func (w *UnlockerWorker) getOTXsFromNonce(ctx context.Context, account string, fromNonce uint64) ([]*store.OTX, error) {
	rows, err := w.wc.store.Pool().Query(ctx, `
		SELECT otx.id, otx.tracking_id, otx.otx_type, keystore.public_key, otx.raw_tx, otx.tx_hash,
		       otx.nonce, otx.replaced, otx.created_at, otx.updated_at, dispatch.status
		FROM keystore
		INNER JOIN otx ON keystore.id = otx.signer_account
		INNER JOIN dispatch ON otx.id = dispatch.otx_id
		WHERE keystore.public_key = $1
		  AND (
		    otx.nonce >= $2
		    OR dispatch.status NOT IN ('SUCCESS', 'REVERTED', 'EXTERNAL_DISPATCH')
		  )
		ORDER BY otx.nonce ASC`, account, fromNonce)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOTXRows(rows)
}

func scanOTXRows(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*store.OTX, error) {
	var otxs []*store.OTX
	for rows.Next() {
		o := &store.OTX{}
		if err := rows.Scan(
			&o.ID, &o.TrackingID, &o.OTXType, &o.SignerAccount,
			&o.RawTx, &o.TxHash, &o.Nonce, &o.Replaced,
			&o.CreatedAt, &o.UpdatedAt, &o.DispatchStatus,
		); err != nil {
			return nil, err
		}
		otxs = append(otxs, o)
	}
	return otxs, rows.Err()
}

func affectedAccounts(otxs []*store.OTX) map[string]struct{} {
	accounts := make(map[string]struct{})
	for _, otx := range otxs {
		accounts[otx.SignerAccount] = struct{}{}
	}
	return accounts
}

func unlockerClassifyRPCError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "nonce too low"):
		return "nonce_low"
	case strings.Contains(msg, "replacement transaction underpriced"):
		return "replacement_underpriced"
	case strings.Contains(msg, "transaction underpriced"), strings.Contains(msg, "gas fee cap is below the minimum base fee"):
		return "gas_price"
	case strings.Contains(msg, "insufficient funds for gas"):
		return "no_gas"
	default:
		return "unknown"
	}
}
