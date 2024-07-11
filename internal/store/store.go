package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
)

type Store interface {
	InsertKeyPair(context.Context, keypair.Key) error
	LoadPrivateKey(context.Context, string) (*ecdsa.PrivateKey, error)
}
