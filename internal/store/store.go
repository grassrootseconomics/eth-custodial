package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Pool() *pgxpool.Pool
	// Keys
	InsertKeyPair(context.Context, pgx.Tx, keypair.Key) error
	LoadPrivateKey(context.Context, pgx.Tx, string) (*ecdsa.PrivateKey, error)
	// Nonce
	PeekNonce(context.Context, pgx.Tx, string) (uint64, error)
	AcquireNonce(context.Context, pgx.Tx, string) (uint64, error)
	SetAccountNonce(context.Context, pgx.Tx, string, uint64) error
	// OTX
	InsertOTX(context.Context, pgx.Tx, string, string) error
}
