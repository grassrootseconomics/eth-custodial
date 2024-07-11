package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
)

func (pg *Pg) InsertKeyPair(ctx context.Context, keypair keypair.Key) error {
	return nil
}

func (pg *Pg) LoadPrivateKey(ctx context.Context, address string) (*ecdsa.PrivateKey, error) {
	return nil, nil
}
