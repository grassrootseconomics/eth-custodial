package store

import (
	"context"
)

func (pg *Pg) PeekNonce(ctx context.Context, address string) (uint64, error) {
	return 0, nil
}

func (pg *Pg) AcquireNonce(ctx context.Context, address string) (uint64, error) {
	return 0, nil
}

func (pg *Pg) SetAccountNonce(ctx context.Context, address string, nonce uint64) error {
	return nil
}
