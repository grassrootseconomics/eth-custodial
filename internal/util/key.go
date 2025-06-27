package util

import (
	"crypto"
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
)

func LoadSigningKey(privateKeyPem string) (crypto.PrivateKey, crypto.PublicKey, error) {
	priv, err := jwt.ParseEdPrivateKeyFromPEM([]byte(privateKeyPem))
	if err != nil {
		return nil, nil, err
	}

	return priv.(ed25519.PrivateKey), priv.(ed25519.PrivateKey).Public().(ed25519.PublicKey), nil
}
