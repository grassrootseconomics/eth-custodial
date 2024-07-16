package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
	"github.com/jackc/pgx/v5"
)

func (pg *Pg) InsertKeyPair(ctx context.Context, tx pgx.Tx, keypair keypair.Key) error {
	return nil
}

func (pg *Pg) LoadPrivateKey(ctx context.Context, tx pgx.Tx, address string) (*ecdsa.PrivateKey, error) {
	return nil, nil
}
