package api

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/kamikazechaser/jrpc"
)

// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_sendtransaction
// DATA, 32 Bytes - the transaction hash, or the zero hash if the transaction is not yet available.
// We ignore other parameters for now as the custodial system takes care of them internally
type sendTransactionParams struct {
	To    string `json:"to"`
	From  string `json:"from"`
	Value string `json:"value"`
	Data  string `json:"data"`
}

func (a *API) methodEthSendTransaction(c jrpc.Context) error {
	var params []sendTransactionParams

	if err := c.Bind(&params); err != nil {
		return err
	}

	a.logg.Debug("eth_sendTransaction request", "body", params)
	return c.Result(common.Hash{}.String())
}
