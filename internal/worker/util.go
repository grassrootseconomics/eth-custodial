package worker

import (
	"errors"
	"math/big"
)

var ErrInvalidNumberString = errors.New("invalid number string")

func StringToBigInt(numberString string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(numberString, 10)
	if !ok {
		return nil, ErrInvalidNumberString
	}

	return n, nil
}
