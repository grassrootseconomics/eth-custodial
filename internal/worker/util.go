package worker

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/grassrootseconomics/ethutils"
)

var ErrInvalidNumberString = errors.New("invalid number string")

func StringToBigInt(numberString string, bump bool) (*big.Int, error) {
	n, ok := new(big.Int).SetString(numberString, 10)
	if !ok {
		return nil, ErrInvalidNumberString
	}

	if bump {
		// To account for demurrage in approval transactions
		// 5%
		bumpMultiplier := big.NewInt(105)
		bumped := new(big.Int).Mul(n, bumpMultiplier)
		bumped.Div(bumped, big.NewInt(100))
		return bumped, nil
	}

	return n, nil
}

func addDivviRefferalTag(p *ethutils.Provider, inputData []byte, user common.Address) []byte {
	referralTag := p.GetReferalTag(user)
	return ethutils.ConcatBytes(inputData, referralTag)
}
