package keypair

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Key struct {
	Public  string `db:"public_key"`
	Private string `db:"private_key"`
}

func GenerateKeyPair() (Key, error) {
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
