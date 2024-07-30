package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Pool() *pgxpool.Pool
	Bootstrap() error
	// Keys
	InsertKeyPair(context.Context, pgx.Tx, keypair.Key) error
	LoadPrivateKey(context.Context, pgx.Tx, string) (*ecdsa.PrivateKey, error)
	LoadMasterSignerKey(context.Context, pgx.Tx) (*ecdsa.PrivateKey, error)
	// Nonce
	PeekNonce(context.Context, pgx.Tx, string) (uint64, error)
	AcquireNonce(context.Context, pgx.Tx, string) (uint64, error)
	SetAccountNonce(context.Context, pgx.Tx, string, uint64) error
	// OTX
	InsertOTX(context.Context, pgx.Tx, string, string) error
}
