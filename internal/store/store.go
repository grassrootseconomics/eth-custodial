package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
)

type Store interface {
	// Keys
	InsertKeyPair(context.Context, keypair.Key) error
	LoadPrivateKey(context.Context, string) (*ecdsa.PrivateKey, error)
	// Nonce
	PeekNonce(context.Context, string) (uint64, error)
	AcquireNonce(context.Context, string) (uint64, error)
	SetAccountNonce(context.Context, string, uint64) error
}
