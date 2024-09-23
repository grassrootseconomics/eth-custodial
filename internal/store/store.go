package store

import (
	"context"

	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Pool() *pgxpool.Pool
	Bootstrap() error
	// Keys
	InsertKeyPair(context.Context, pgx.Tx, keypair.Key) error
	CheckKeypair(context.Context, pgx.Tx, string) (bool, error)
	LoadPrivateKey(context.Context, pgx.Tx, string) (keypair.Key, error)
	LoadMasterSignerKey(context.Context, pgx.Tx) (keypair.Key, error)
	// Nonce
	PeekNonce(context.Context, pgx.Tx, string) (uint64, error)
	AcquireNonce(context.Context, pgx.Tx, string) (uint64, error)
	SetAccountNonce(context.Context, pgx.Tx, string, uint64) error
	// OTX
	InsertOTX(context.Context, pgx.Tx, OTX) error
	GetOTXByTrackingID(context.Context, pgx.Tx, string) (OTX, error)
	GetOTXByAccount(context.Context, pgx.Tx, string, int) ([]OTX, error)
	GetOTXByAccountNext(context.Context, pgx.Tx, string, int, int) ([]OTX, error)
	GetOTXByAccountPrevious(context.Context, pgx.Tx, string, int, int) ([]OTX, error)
}
