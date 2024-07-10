package keystore

import (
	"context"
	"crypto/ecdsa"
)

type Keystore interface {
	LoadPrivateKey(context.Context, string) (*ecdsa.PrivateKey, error)
	CreateKeyPair(context.Context, Key) (uint, error)
}
