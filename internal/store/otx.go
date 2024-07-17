package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (pg *Pg) InsertOTX(ctx context.Context, tx pgx.Tx, txHash string, rawTx string) error {
	return nil
}
