package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type (
	OTX struct {
		ID            uint64    `db:"id" json:"id"`
		TrackingID    string    `db:"tracking_id" json:"trackingId"`
		OTXType       string    `db:"otx_type" json:"otxType"`
		SignerAccount string    `db:"public_address" json:"signerAccount"`
		RawTx         string    `db:"raw_tx" json:"rawTx"`
		TxHash        string    `db:"tx_hash" json:"txHash"`
		Nonce         uint64    `db:"nonce" json:"nonce"`
		Replaced      string    `db:"replaced" json:"replaced"`
		CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	}
)

const (
	GAS_REFILL       string = "REFILL_GAS"
	ACCOUNT_REGISTER string = "ACCOUNT_REGISTER"
	GAS_TRANSFER     string = "GAS_TRANSFER"
	TOKEN_TRANSFER   string = "TOKEN_TRANSFER"
	TOKEN_APPROVE    string = "TOKEN_APPROVE"
	POOL_SWAP        string = "POOL_SWAP"
	POOL_DEPSOIT     string = "POOL_DESPOSIT"
	OTHER_MANUAL     string = "OTHER_MANUAL"
)

func (pg *Pg) InsertOTX(ctx context.Context, tx pgx.Tx, otx OTX) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.InsertOTX,
		otx.TrackingID,
		otx.OTXType,
		otx.SignerAccount,
		otx.RawTx,
		otx.TxHash,
		otx.Nonce,
	)
	if err != nil {
		return err
	}

	return nil
}

func (pg *Pg) GetOTXByTrackingID(ctx context.Context, tx pgx.Tx, trackingID string) (OTX, error) {
	return OTX{}, nil
}

func (pg *Pg) GetOTXByAccount(ctx context.Context, tx pgx.Tx, trackingID string, limit int) ([]OTX, error) {
	return nil, nil
}

func (pg *Pg) GetOTXByAccountNext(ctx context.Context, tx pgx.Tx, trackingID string, cursor int, limit int) ([]OTX, error) {
	return nil, nil
}

func (pg *Pg) GetOTXByAccountPrevious(ctx context.Context, tx pgx.Tx, trackingID string, cursor int, limit int) ([]OTX, error) {
	return nil, nil
}
