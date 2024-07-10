package keystore

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Key struct {
	Public  string
	Private string
}

func generateKeyPair() (Key, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return Key{}, err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	publicKey := privateKey.Public()
	publicKeyECDSA := publicKey.(*ecdsa.PublicKey)

	return Key{
		Public:  crypto.PubkeyToAddress(*publicKeyECDSA).Hex(),
		Private: hexutil.Encode(privateKeyBytes)[2:],
	}, nil
}
