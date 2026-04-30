package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/knadh/koanf/v2"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

var (
	confFlag       string
	queriesFlag    string
	migrationsFlag string
	rpcFlag        string
	dryRun         bool

	lo *slog.Logger
	ko *koanf.Koanf
)

func init() {
	flag.StringVar(&confFlag, "config", "config.toml", "Config file location")
	flag.StringVar(&queriesFlag, "queries", "queries.sql", "Queries file location")
	flag.StringVar(&migrationsFlag, "migrations", "migrations/", "Migrations folder location")
	flag.StringVar(&rpcFlag, "rpc", "", "Comma-separated RPC endpoints (overrides config)")
	flag.BoolVar(&dryRun, "dry-run", false, "Log actions without executing")
}

func main() {
	flag.Parse()

	lo = util.InitLogger()
	ko = util.InitConfig(lo, confFlag)

	ctx := context.Background()

	pgStore, err := store.NewPgStore(store.PgOpts{
		Logg:                 lo,
		DSN:                  ko.MustString("postgres.dsn"),
		MigrationsFolderPath: migrationsFlag,
		QueriesFolderPath:    queriesFlag,
	})
	if err != nil {
		lo.Error("failed to initialize store", "error", err)
		os.Exit(1)
	}

	endpoints := parseRPCEndpoints()
	if len(endpoints) == 0 {
		lo.Error("no RPC endpoints configured")
		os.Exit(1)
	}
	lo.Info("rpc endpoints loaded", "count", len(endpoints))

	clients := make([]*w3.Client, 0, len(endpoints))
	for _, ep := range endpoints {
		c, err := w3.Dial(ep)
		if err != nil {
			lo.Warn("failed to dial RPC endpoint", "endpoint", ep, "error", err)
			continue
		}
		defer c.Close()
		clients = append(clients, c)
	}
	if len(clients) == 0 {
		lo.Error("could not connect to any RPC endpoint")
		os.Exit(1)
	}

	chainID := big.NewInt(ko.MustInt64("chain.id"))
	signer := types.LatestSignerForChainID(chainID)

	stuckOTXs, err := getStuckOTXs(ctx, pgStore)
	if err != nil {
		lo.Error("failed to get stuck OTXs", "error", err)
		os.Exit(1)
	}

	if len(stuckOTXs) == 0 {
		lo.Info("no stuck transactions older than 5 minutes found")
		return
	}
	lo.Info("found stuck transactions", "count", len(stuckOTXs))

	affectedAccounts := affectedAccountSet(stuckOTXs)
	noGasAccounts := make(map[string]struct{})

	var stats struct {
		resubmitted int
		resigned    int
		alreadyDone int
		skipped     int
		failed      int
	}

	for account := range affectedAccounts {
		networkNonce, err := getNonceMultiNode(ctx, clients, common.HexToAddress(account))
		if err != nil {
			lo.Error("failed to get network nonce, skipping account", "account", account, "error", err)
			stats.skipped++
			continue
		}
		lo.Info("processing account", "account", account, "network_nonce", networkNonce)

		// Fetch all OTXs from the network nonce onwards not just the stuck ones.
		// The stuck list may start at nonce 1195 while the node is at 1191, meaning
		// 1191-1194 exist in otx but weren't flagged as stuck (wrong status/too fresh).
		otxs, err := getOTXsFromNonce(ctx, pgStore, account, networkNonce)
		if err != nil {
			lo.Error("failed to fetch OTXs from nonce, skipping account", "account", account, "error", err)
			stats.skipped++
			continue
		}

		if len(otxs) == 0 {
			lo.Info("no OTXs to process for account", "account", account)
			continue
		}
		lo.Info("OTXs to resubmit", "account", account, "count", len(otxs), "from_nonce", otxs[0].Nonce)

		skipAccount := false
		for _, otx := range otxs {
			if skipAccount {
				stats.skipped++
				continue
			}

			lo.Info("processing OTX",
				"otx_id", otx.ID,
				"nonce", otx.Nonce,
				"status", otx.DispatchStatus,
				"tx_hash", otx.TxHash,
				"type", otx.OTXType,
			)

			if otx.Nonce < networkNonce {
				if err := checkReceiptAndUpdate(ctx, clients, pgStore, otx); err != nil {
					lo.Warn("failed to check receipt for already-mined tx", "otx_id", otx.ID, "error", err)
				}
				stats.alreadyDone++
				continue
			}

			if dryRun {
				lo.Info("[DRY RUN] would resubmit", "otx_id", otx.ID, "nonce", otx.Nonce)
				continue
			}

			rawTxBytes, err := hexutil.Decode(otx.RawTx)
			if err != nil {
				lo.Error("failed to decode raw tx", "otx_id", otx.ID, "error", err)
				stats.failed++
				continue
			}

			err = sendRawTxMultiNode(ctx, clients, rawTxBytes)
			if err == nil {
				lo.Info("resubmitted successfully", "otx_id", otx.ID, "nonce", otx.Nonce)
				if err := updateDispatchStatus(ctx, pgStore, otx.ID, store.IN_NETWORK); err != nil {
					lo.Error("failed to update dispatch status", "otx_id", otx.ID, "error", err)
				}
				stats.resubmitted++
				continue
			}

			errType := classifyRPCError(err)
			lo.Info("resubmit failed, classifying", "otx_id", otx.ID, "error_type", errType, "error", err)

			switch errType {
			case "nonce_low":
				if err := checkReceiptAndUpdate(ctx, clients, pgStore, otx); err != nil {
					lo.Warn("failed to check receipt", "otx_id", otx.ID, "error", err)
				}
				stats.alreadyDone++

			case "gas_price", "replacement_underpriced":
				lo.Info("gas-related error, attempting re-sign", "otx_id", otx.ID)

				newRawTxHex, newTxHash, err := resignAndSubmit(ctx, clients, pgStore, signer, otx, account)
				if err != nil {
					if classifyRPCError(err) == "no_gas" {
						lo.Warn("re-sign requires gas top-up, skipping remaining txs", "account", account, "otx_id", otx.ID)
						noGasAccounts[account] = struct{}{}
						skipAccount = true
						stats.skipped++
						continue
					}
					lo.Error("re-sign failed", "otx_id", otx.ID, "error", err)
					stats.failed++
					continue
				}

				if err := updateOTXAndDispatch(ctx, pgStore, otx.ID, newRawTxHex, newTxHash); err != nil {
					lo.Error("failed to update OTX after re-sign", "otx_id", otx.ID, "error", err)
					stats.failed++
					continue
				}
				lo.Info("re-signed and resubmitted", "otx_id", otx.ID, "new_tx_hash", newTxHash)
				stats.resigned++

			case "no_gas":
				lo.Warn("account has insufficient gas, skipping remaining txs", "account", account)
				noGasAccounts[account] = struct{}{}
				skipAccount = true
				stats.skipped++

			default:
				lo.Error("unhandled error", "otx_id", otx.ID, "error", err)
				stats.failed++
			}
		}
	}

	lo.Info("unlocker complete",
		"resubmitted", stats.resubmitted,
		"resigned", stats.resigned,
		"already_mined", stats.alreadyDone,
		"skipped", stats.skipped,
		"failed", stats.failed,
	)

	if len(noGasAccounts) > 0 {
		lo.Warn("accounts requiring gas top-up detected", "count", len(noGasAccounts))
		if err := printGasTopupCSV(os.Stdout, noGasAccounts, "0.5"); err != nil {
			lo.Error("failed to print gas top-up csv", "error", err)
		}
	}
}

func parseRPCEndpoints() []string {
	if rpcFlag != "" {
		parts := strings.Split(rpcFlag, ",")
		var endpoints []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				endpoints = append(endpoints, p)
			}
		}
		return endpoints
	}

	return []string{ko.MustString("chain.rpc_endpoint")}
}

func getStuckOTXs(ctx context.Context, pgStore store.Store) ([]*store.OTX, error) {
	const q = `
		SELECT otx.id, otx.tracking_id, otx.otx_type, keystore.public_key, otx.raw_tx, otx.tx_hash,
		       otx.nonce, otx.replaced, otx.created_at, otx.updated_at, dispatch.status
		FROM keystore
		INNER JOIN otx ON keystore.id = otx.signer_account
		INNER JOIN dispatch ON otx.id = dispatch.otx_id
		WHERE dispatch.status NOT IN ('SUCCESS', 'REVERTED', 'PENDING', 'EXTERNAL_DISPATCH')
		  AND otx.otx_type NOT IN ('GENERIC_SIGN', 'OTHER_MANUAL')
		  AND dispatch.updated_at <= NOW() - INTERVAL '5 minutes'
		ORDER BY otx.id ASC
		LIMIT 100`

	rows, err := pgStore.Pool().Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func affectedAccountSet(otxs []*store.OTX) map[string]struct{} {
	accounts := make(map[string]struct{})
	for _, otx := range otxs {
		accounts[otx.SignerAccount] = struct{}{}
	}
	return accounts
}

func printGasTopupCSV(w io.Writer, accounts map[string]struct{}, amount string) error {
	for _, account := range sortedAccounts(accounts) {
		if _, err := fmt.Fprintf(w, "%s,%s\n", account, amount); err != nil {
			return err
		}
	}
	return nil
}

func sortedAccounts(accounts map[string]struct{}) []string {
	list := make([]string, 0, len(accounts))
	for account := range accounts {
		list = append(list, account)
	}
	sort.Strings(list)
	return list
}

// getOTXsFromNonce returns all OTXs for an account with nonce >= fromNonce, ordered by nonce ASC.
// This is intentionally broader than getStuckOTXs — it includes any status so that nonces the
// node hasn't seen yet (e.g. 1191-1194) are always included even if their dispatch record looks fine.
func getOTXsFromNonce(ctx context.Context, pgStore store.Store, account string, fromNonce uint64) ([]*store.OTX, error) {
	const q = `
		SELECT otx.id, otx.tracking_id, otx.otx_type, keystore.public_key, otx.raw_tx, otx.tx_hash,
		       otx.nonce, otx.replaced, otx.created_at, otx.updated_at, dispatch.status
		FROM keystore
		INNER JOIN otx ON keystore.id = otx.signer_account
		INNER JOIN dispatch ON otx.id = dispatch.otx_id
		WHERE keystore.public_key = $1
		  AND otx.nonce >= $2
		ORDER BY otx.nonce ASC`

	rows, err := pgStore.Pool().Query(ctx, q, account, fromNonce)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func getNonceMultiNode(ctx context.Context, clients []*w3.Client, addr common.Address) (uint64, error) {
	var lastErr error
	for _, c := range clients {
		var nonce uint64
		if err := c.CallCtx(ctx, eth.Nonce(addr, nil).Returns(&nonce)); err != nil {
			lastErr = err
			continue
		}
		return nonce, nil
	}
	return 0, fmt.Errorf("all nodes failed to get nonce: %w", lastErr)
}

func sendRawTxMultiNode(ctx context.Context, clients []*w3.Client, rawTx []byte) error {
	var lastErr error
	for _, c := range clients {
		var txHash common.Hash
		var callErrs w3.CallErrors

		if err := c.CallCtx(ctx, eth.SendRawTx(rawTx).Returns(&txHash)); errors.As(err, &callErrs) {
			if _, ok := callErrs[0].(rpc.Error); ok {
				return callErrs[0]
			}
			return err
		} else if err != nil {
			if isNetworkError(err) {
				lastErr = err
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("all nodes failed: %w", lastErr)
}

func getGasPriceMultiNode(ctx context.Context, clients []*w3.Client) (*big.Int, *big.Int, error) {
	var lastErr error
	for _, c := range clients {
		var gasPrice, tipCap *big.Int
		if err := c.CallCtx(ctx,
			eth.GasPrice().Returns(&gasPrice),
			eth.GasTipCap().Returns(&tipCap),
		); err != nil {
			lastErr = err
			continue
		}
		// Bump by 20% to accommodate fluctuations
		bumpFactor := big.NewInt(120)
		gasPrice.Mul(gasPrice, bumpFactor)
		gasPrice.Div(gasPrice, big.NewInt(100))
		return gasPrice, tipCap, nil
	}
	return nil, nil, fmt.Errorf("all nodes failed to get gas price: %w", lastErr)
}

func getReceiptMultiNode(ctx context.Context, clients []*w3.Client, txHash common.Hash) (*types.Receipt, error) {
	var lastErr error
	for _, c := range clients {
		var receipt *types.Receipt
		if err := c.CallCtx(ctx, eth.TxReceipt(txHash).Returns(&receipt)); err != nil {
			lastErr = err
			continue
		}
		return receipt, nil
	}
	return nil, fmt.Errorf("all nodes failed to get receipt: %w", lastErr)
}

func checkReceiptAndUpdate(ctx context.Context, clients []*w3.Client, pgStore store.Store, otx *store.OTX) error {
	receipt, err := getReceiptMultiNode(ctx, clients, common.HexToHash(otx.TxHash))
	if err != nil {
		return err
	}

	status := store.REVERTED
	if receipt.Status == 1 {
		status = store.SUCCESS
	}

	return updateDispatchStatus(ctx, pgStore, otx.ID, status)
}

func resignAndSubmit(
	ctx context.Context,
	clients []*w3.Client,
	pgStore store.Store,
	chainSigner types.Signer,
	otx *store.OTX,
	account string,
) (string, string, error) {
	originalTxBytes, err := hexutil.Decode(otx.RawTx)
	if err != nil {
		return "", "", fmt.Errorf("decode raw tx: %w", err)
	}

	originalTx := new(types.Transaction)
	if err := originalTx.UnmarshalBinary(originalTxBytes); err != nil {
		return "", "", fmt.Errorf("unmarshal tx: %w", err)
	}

	newGasFeeCap, newGasTipCap, err := getGasPriceMultiNode(ctx, clients)
	if err != nil {
		return "", "", fmt.Errorf("get gas price: %w", err)
	}

	// Ensure new gas price is higher than original to satisfy replacement rules
	if originalTx.GasFeeCap() != nil && newGasFeeCap.Cmp(originalTx.GasFeeCap()) <= 0 {
		bump := new(big.Int).Mul(originalTx.GasFeeCap(), big.NewInt(115))
		bump.Div(bump, big.NewInt(100))
		newGasFeeCap = bump
	}
	if originalTx.GasTipCap() != nil && newGasTipCap.Cmp(originalTx.GasTipCap()) <= 0 {
		bump := new(big.Int).Mul(originalTx.GasTipCap(), big.NewInt(115))
		bump.Div(bump, big.NewInt(100))
		newGasTipCap = bump
	}

	dbTx, err := pgStore.Pool().Begin(ctx)
	if err != nil {
		return "", "", fmt.Errorf("begin tx: %w", err)
	}
	defer dbTx.Rollback(ctx)

	keypair, err := pgStore.LoadPrivateKey(ctx, dbTx, account)
	if err != nil {
		return "", "", fmt.Errorf("load private key: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(keypair.Private)
	if err != nil {
		return "", "", fmt.Errorf("parse private key: %w", err)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return "", "", fmt.Errorf("commit read tx: %w", err)
	}

	newRawTxHex, newTxHash, err := resignTx(originalTx, privateKey, chainSigner, newGasFeeCap, newGasTipCap)
	if err != nil {
		return "", "", fmt.Errorf("re-sign: %w", err)
	}

	newRawTxBytes, err := hexutil.Decode(newRawTxHex)
	if err != nil {
		return "", "", fmt.Errorf("decode new raw tx: %w", err)
	}

	if err := sendRawTxMultiNode(ctx, clients, newRawTxBytes); err != nil {
		return "", "", fmt.Errorf("submit re-signed tx: %w", err)
	}

	return newRawTxHex, newTxHash, nil
}

func resignTx(
	originalTx *types.Transaction,
	privateKey *ecdsa.PrivateKey,
	signer types.Signer,
	newGasFeeCap *big.Int,
	newGasTipCap *big.Int,
) (string, string, error) {
	to := originalTx.To()
	if to == nil {
		return "", "", fmt.Errorf("original tx has no recipient (contract creation)")
	}

	newTx, err := types.SignNewTx(privateKey, signer, &types.DynamicFeeTx{
		Nonce:     originalTx.Nonce(),
		To:        to,
		Value:     originalTx.Value(),
		Data:      originalTx.Data(),
		Gas:       originalTx.Gas(),
		GasFeeCap: newGasFeeCap,
		GasTipCap: newGasTipCap,
	})
	if err != nil {
		return "", "", fmt.Errorf("sign new tx: %w", err)
	}

	rawTxBytes, err := newTx.MarshalBinary()
	if err != nil {
		return "", "", fmt.Errorf("marshal new tx: %w", err)
	}

	return hexutil.Encode(rawTxBytes), newTx.Hash().Hex(), nil
}

func updateDispatchStatus(ctx context.Context, pgStore store.Store, otxID uint64, status string) error {
	dbTx, err := pgStore.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	if err := pgStore.UpdateDispatchTxStatus(ctx, dbTx, store.DispatchTx{
		OTXID:  otxID,
		Status: status,
	}); err != nil {
		return err
	}

	return dbTx.Commit(ctx)
}

func updateOTXAndDispatch(ctx context.Context, pgStore store.Store, otxID uint64, newRawTxHex string, newTxHash string) error {
	dbTx, err := pgStore.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	if _, err := dbTx.Exec(ctx,
		`UPDATE otx SET raw_tx = $1, tx_hash = $2 WHERE id = $3`,
		newRawTxHex, newTxHash, otxID,
	); err != nil {
		return err
	}

	if err := pgStore.UpdateDispatchTxStatus(ctx, dbTx, store.DispatchTx{
		OTXID:  otxID,
		Status: store.IN_NETWORK,
	}); err != nil {
		return err
	}

	return dbTx.Commit(ctx)
}

func classifyRPCError(err error) string {
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
	case isNetworkError(err):
		return "network"
	default:
		return "unknown"
	}
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	var urlErr *url.Error

	return errors.As(err, &netErr) ||
		errors.As(err, &urlErr) ||
		strings.Contains(err.Error(), "timeout") ||
		errors.Is(err, context.DeadlineExceeded)
}
