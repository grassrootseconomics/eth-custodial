package worker

type (
	GasTransferPayload struct {
		TrackingId string `json:"trackingId"`
		From       string `json:"from"`
		To         string `json:"to"`
		Amount     uint64 `json:"amount"`
	}

	SwapPoolWithdrawPayload struct {
		TrackingId       string `json:"trackingId"`
		PoolAddress      string `json:"poolAddress"`
		FromTokenAddress string `json:"fromTokenAddress"`
		FromAmount       int64  `json:"fromAmount"`
		ToTokenAddress   string `json:"toTokenAddress"`
		ToAmount         uint64 `json:"toAmount"`
	}

	SwapPoolSeedPayload struct {
		TrackingId  string `json:"trackingId"`
		PoolAddress string `json:"poolAddress"`
		Token       string `json:"tokenAddress"`
		Amount      uint64 `json:"amount"`
	}

	TokenApprovePayload struct {
		TrackingId        string `json:"trackingId"`
		Amount            uint64 `json:"amount"`
		Authorizer        string `json:"authorizer"`
		AuthorizedAddress string `json:"authorizedAddress"`
		TokenAddress      string `json:"tokenAddress"`
	}
)
