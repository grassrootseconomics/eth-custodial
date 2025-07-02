package store

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

type OTX struct {
	ID             uint64    `db:"id" json:"id"`
	TrackingID     string    `db:"tracking_id" json:"trackingId"`
	OTXType        string    `db:"otx_type" json:"otxType"`
	SignerAccount  string    `db:"public_key" json:"signerAccount"`
	RawTx          string    `db:"raw_tx" json:"rawTx"`
	TxHash         string    `db:"tx_hash" json:"txHash"`
	Nonce          uint64    `db:"nonce" json:"nonce"`
	Replaced       bool      `db:"replaced" json:"replaced"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
	DispatchStatus string    `db:"status" json:"status"`
}

const (
	GAS_REFILL            string = "GAS_REFILL"
	ACCOUNT_REGISTER      string = "ACCOUNT_REGISTER"
	GAS_TRANSFER          string = "GAS_TRANSFER"
	TOKEN_TRANSFER        string = "TOKEN_TRANSFER"
	TOKEN_APPROVE         string = "TOKEN_APPROVE"
	TOKEN_SWEEP           string = "TOKEN_SWEEP"
	POOL_SWAP             string = "POOL_SWAP"
	POOL_DEPOSIT          string = "POOL_DEPOSIT"
	OTHER_MANUAL          string = "OTHER_MANUAL"
	GENERIC_SIGN          string = "GENERIC_SIGN"
	STANDARD_TOKEN_DEPLOY string = "STANDARD_TOKEN_DEPLOY"
	TRANSFER_OWNERSHIP    string = "TRANSFER_OWNERSHIP"
	TOKEN_INDEX_ADD       string = "TOKEN_INDEX_ADD"
)

func (pg *Pg) InsertOTX(ctx context.Context, tx pgx.Tx, otx OTX) (uint64, error) {
	var id uint64

	if err := tx.QueryRow(
		ctx,
		pg.queries.InsertOTX,
		otx.TrackingID,
		otx.OTXType,
		otx.SignerAccount,
		otx.RawTx,
		otx.TxHash,
		otx.Nonce,
	).Scan(&id); err != nil {
		return id, err
	}

	return id, nil
}

func (pg *Pg) GetOTXByTxHash(ctx context.Context, tx pgx.Tx, txHash string) (OTX, error) {
	var otx OTX

	row, err := tx.Query(ctx, pg.queries.GetOTXByTxHash, txHash)
	if err != nil {
		return otx, err
	}

	if err := pgxscan.ScanOne(&otx, row); err != nil {
		return otx, err
	}

	return otx, nil
}

func (pg *Pg) GetOTXByTrackingID(ctx context.Context, tx pgx.Tx, trackingID string) ([]*OTX, error) {
	var otx []*OTX

	if err := pgxscan.Select(ctx, tx, &otx, pg.queries.GetOTXByTrackingID, trackingID); err != nil {
		return otx, err
	}

	return otx, nil
}

func (pg *Pg) GetOTXByAccount(ctx context.Context, tx pgx.Tx, publicKey string, limit int) ([]*OTX, error) {
	var otx []*OTX

	if err := pgxscan.Select(ctx, tx, &otx, pg.queries.GetOTXByAccount, publicKey, limit); err != nil {
		return nil, err
	}

	return otx, nil
}

func (pg *Pg) GetOTXByAccountNext(ctx context.Context, tx pgx.Tx, publicKey string, cursor int, limit int) ([]*OTX, error) {
	var otx []*OTX

	if err := pgxscan.Select(ctx, tx, &otx, pg.queries.GetOTXByAccountNext, publicKey, cursor, limit); err != nil {
		return nil, err
	}

	return otx, nil
}

func (pg *Pg) GetOTXByAccountPrevious(ctx context.Context, tx pgx.Tx, publicKey string, cursor int, limit int) ([]*OTX, error) {
	var otx []*OTX

	if err := pgxscan.Select(ctx, tx, &otx, pg.queries.GetOTXByAccountPrevious, publicKey, cursor, limit); err != nil {
		return nil, err
	}

	return otx, nil
}

func (pg *Pg) GetFailedOTX(ctx context.Context, tx pgx.Tx) ([]*OTX, error) {
	var otx []*OTX

	if err := pgxscan.Select(ctx, tx, &otx, pg.queries.GetFailedOTX); err != nil {
		return nil, err
	}

	return otx, nil
}
