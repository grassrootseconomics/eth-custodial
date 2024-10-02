package worker

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/grassrootseconomics/ethutils"
)

var publicKey = ethutils.HexToAddress("0x313b295C92f43960aaA019CCFDd75bECdd355Df9")

// Tested against a live RPC endpoint
func TestDisptachWorker_sendRawTx(t *testing.T) {
	// throwaway key
	// 0x313b295C92f43960aaA019CCFDd75bECdd355Df9
	privateKey := "f4d7cf5f04545f2b9f7a3acc3e91769bdc1026955e8b2659c5da849878126733"

	successfulScenario := ethutils.GasTransferTxOpts{
		To:        ethutils.ZeroAddress,
		Value:     big.NewInt(0),
		GasFeeCap: big.NewInt(300000000000),
		GasTipCap: big.NewInt(10000000000),
		Nonce:     0,
	}

	type args struct {
		chainID     int64
		rpcEndpoint string
		tx          *types.Transaction
		wantErr     bool
		errType     error
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &DisptachWorker{
				WorkerDefaults: tt.fields.WorkerDefaults,
				wc:             tt.fields.wc,
			}
			if err := w.sendRawTx(tt.args.ctx, tt.args.rawTx); (err != nil) != tt.wantErr {
				t.Errorf("DisptachWorker.sendRawTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getNonce(chainProvider *ethutils.Provider, account common.Address) uint64 {
	return 0
}
