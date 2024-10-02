package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type DispatchTx struct {
	ID        uint64    `db:"id" json:"id"`
	OTXID     uint64    `db:"otx_id" json:"otxId"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

const (
	PENDING           string = "PENDING"
	IN_NETWORK        string = "IN_NETWORK"
	SUCCESS           string = "SUCCESS"
	REVERTED          string = "REVERTED"
	LOW_NONCE         string = "LOW_NONCE"
	NO_GAS            string = "NO_GAS"
	LOW_GAS_PRICE     string = "LOW_GAS_PRICE"
	NETWORK_ERROR     string = "NETWORK_ERROR"
	EXTERNAL_DISPATCH string = "EXTERNAL_DISPATCH"
	UNKNOWN_RPC_ERROR string = "UNKNOWN_RPC_ERROR"
)

func (pg *Pg) InsertDispatchTx(ctx context.Context, tx pgx.Tx, dispatchTx DispatchTx) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.InsertDispatchTx,
		dispatchTx.OTXID,
		dispatchTx.Status,
	)
	if err != nil {
		return err
	}

	return nil
}

func (pg *Pg) UpdateDispatchTxStatus(ctx context.Context, tx pgx.Tx, dispatchTx DispatchTx) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.UpdateDispatchTxStatus,
		dispatchTx.Status,
		dispatchTx.OTXID,
	)
	if err != nil {
		return err
	}

	return nil
}
