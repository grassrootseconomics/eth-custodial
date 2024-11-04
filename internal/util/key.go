package util

import (
	"crypto"
	"crypto/ed25519"

	"github.com/golang-jwt/jwt"
)

func LoadSigningKey(privateKeyPem string) (crypto.PrivateKey, crypto.PublicKey, error) {
	priv, err := jwt.ParseEdPrivateKeyFromPEM([]byte(privateKeyPem))
	if err != nil {
		return nil, nil, err
	}

	return priv.(ed25519.PrivateKey), priv.(ed25519.PrivateKey).Public().(ed25519.PublicKey), nil

	// block, _ := pem.Decode([]byte(privateKeyPem))
	// if block == nil {
	// 	return nil, nil, errors.New("failed to decode PEM block containing private key")
	// }

	// privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	// if err != nil {
	// 	return nil, nil, err
	// }

	// privateKey = privateKey.(ed25519.PrivateKey)
	// publicKey := privateKey.(ed25519.PrivateKey).Public().(ed25519.PublicKey)

	// return privateKey, publicKey, nil
}
