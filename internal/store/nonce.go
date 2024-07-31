package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (pg *Pg) PeekNonce(ctx context.Context, tx pgx.Tx, publicKey string) (uint64, error) {
	var nonce uint64

	if err := tx.QueryRow(
		ctx,
		pg.queries.PeekNonce,
		publicKey,
	).Scan(&nonce); err != nil {
		return 0, err
	}

	return nonce, nil
}

func (pg *Pg) AcquireNonce(ctx context.Context, tx pgx.Tx, publicKey string) (uint64, error) {
	var nonce uint64

	if err := tx.QueryRow(
		ctx,
		pg.queries.AcquireNonce,
		publicKey,
	).Scan(&nonce); err != nil {
		return 0, err
	}

	return nonce, nil
}

func (pg *Pg) SetAccountNonce(ctx context.Context, tx pgx.Tx, publicKey string, nonce uint64) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.SetAcccountNonce,
		publicKey,
		nonce,
	)
	if err != nil {
		return err
	}

	return nil
}
