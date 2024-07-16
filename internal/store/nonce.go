package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (pg *Pg) PeekNonce(ctx context.Context, tx pgx.Tx, address string) (uint64, error) {
	return 0, nil
}

func (pg *Pg) AcquireNonce(ctx context.Context, tx pgx.Tx, address string) (uint64, error) {
	return 0, nil
}

func (pg *Pg) SetAccountNonce(ctx context.Context, tx pgx.Tx, address string, nonce uint64) error {
	return nil
}
