package worker

import (
	"math/big"
	"testing"

	"github.com/lmittmann/w3"
)

func Test_decodeTx(t *testing.T) {
	// CEL2 Alfajores: 0xafe423688373e8da4bc2ff86fa8c120bb3a7ab4e18a4046eeaa17f50b824e069
	rawTx := "0x02f8b282aef380830f42408506fc35fb80830557309493bb5f14464a9b7e5d5487dab12d100417f2332380b844a9059cbb0000000000000000000000009cbcd1c2e587c8ecd8ab05a33d28a6c438a2adec00000000000000000000000000000000000000000000000000000000004c4b40c001a04518d8223d648c8be464945dd9630fa9ac995d93e1274a9f07bdae4d907e25d0a06724c3917171acd4e1f23414bbc29c42ca4273d573b848229ea96746a928e925"
	tx, err := decodeTx(rawTx)
	if err != nil {
		t.Errorf("decodeTx() error = %v", err)
		return
	}
	if tx == nil {
		t.Errorf("decodeTx() got nil tx")
	}
}

func Test_bumpGas(t *testing.T) {
	// CEL2 Alfajores: 0xafe423688373e8da4bc2ff86fa8c120bb3a7ab4e18a4046eeaa17f50b824e069
	// We know the original gas fee cap was 30001200000

	rawTx := "0x02f8b282aef380830f42408506fc35fb80830557309493bb5f14464a9b7e5d5487dab12d100417f2332380b844a9059cbb0000000000000000000000009cbcd1c2e587c8ecd8ab05a33d28a6c438a2adec00000000000000000000000000000000000000000000000000000000004c4b40c001a04518d8223d648c8be464945dd9630fa9ac995d93e1274a9f07bdae4d907e25d0a06724c3917171acd4e1f23414bbc29c42ca4273d573b848229ea96746a928e925"
	bumpedTx, err := bumpGas(rawTx)
	if err != nil {
		t.Errorf("decodeTx() error = %v", err)
		return
	}
	if !gt(bumpedTx.GasFeeCap, big.NewInt(30001200000)) {
		t.Errorf("bumpGas() got = %v, want = %v", bumpedTx.GasFeeCap, w3.I("25 gwei"))
	}
}

func gt(a, b *big.Int) bool {
	if a.Cmp(b) > 0 {
		return true
	}
	return false
}
