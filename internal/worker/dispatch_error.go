package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInsufficientGas          = errors.New("eth-custodial: insufficient gas")
	ErrGasPriceTooLow           = errors.New("eth-custodial: gas price too low")
	ErrNonceTooLow              = errors.New("eth-custodial: nonce too low")
	ErrReplacementTxUnderpriced = errors.New("eth-custodial: replacement tx underpriced")
	ErrNetwork                  = errors.New("eth-custodial: network related error")
)

type DispatchError struct {
	Err         error
	OriginalErr error
}

func (e *DispatchError) Error() string {
	return fmt.Sprintf("%v (original rpc error: %v)", e.Err, e.OriginalErr)
}

func (e *DispatchError) Unwrap() error {
	return e.Err
}

func handleJSONRPCError(errMsg string) error {
	switch {
	case strings.Contains(errMsg, "insufficient funds for gas"):
		return ErrInsufficientGas
	case strings.Contains(errMsg, "transaction underpriced"):
		return ErrGasPriceTooLow
	case strings.Contains(errMsg, "nonce too low"):
		return ErrNonceTooLow
	case strings.Contains(errMsg, "replacement transaction underpriced"):
		return ErrReplacementTxUnderpriced
	default:
		return nil
	}
}

func handleNetworkError(err error) error {
	if err == nil {
		return nil
	}

	var netErr net.Error
	var urlErr *url.Error

	if errors.As(err, &netErr) || errors.As(err, &urlErr) || strings.Contains(err.Error(), "timeout") || errors.Is(err, context.DeadlineExceeded) {
		return &DispatchError{
			Err:         ErrNetwork,
			OriginalErr: err,
		}
	}
	return err
}
