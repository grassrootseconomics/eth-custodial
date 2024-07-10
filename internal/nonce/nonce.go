package nonce

import "context"

type Noncestore interface {
	Peek(context.Context, string) (uint64, error)
	Acquire(context.Context, string) (uint64, error)
	SetAccountNonce(context.Context, string, uint64) error
}
