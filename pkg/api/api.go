package api

type (
	OKResponse struct {
		Ok          bool           `json:"ok"`
		Description string         `json:"description"`
		Result      map[string]any `json:"result"`
	}

	ErrResponse struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
		ErrCode     string `json:"errorCode"`
	}

	LoginRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	TransferRequest struct {
		From         string `json:"from" validate:"required,eth_addr_checksum"`
		To           string `json:"to" validate:"required,eth_addr_checksum"`
		TokenAddress string `json:"tokenAddress" validate:"required,eth_addr_checksum"`
		Amount       string `json:"amount" validate:"required,number,gt=0"`
	}

	SweepRequest struct {
		From         string `json:"from" validate:"required,eth_addr_checksum"`
		To           string `json:"to" validate:"required,eth_addr_checksum"`
		TokenAddress string `json:"tokenAddress" validate:"required,eth_addr_checksum"`
	}

	PoolSwapRequest struct {
		From             string `json:"from" validate:"required,eth_addr_checksum"`
		FromTokenAddress string `json:"fromTokenAddress" validate:"required,eth_addr_checksum"`
		ToTokenAddress   string `json:"toTokenAddress" validate:"required,eth_addr_checksum"`
		PoolAddress      string `json:"poolAddress" validate:"required,eth_addr_checksum"`
		Amount           string `json:"amount" validate:"required,number,gt=0"`
	}

	PoolDepositRequest struct {
		From         string `json:"from" validate:"required,eth_addr_checksum"`
		TokenAddress string `json:"tokenAddress" validate:"required,eth_addr_checksum"`
		PoolAddress  string `json:"poolAddress" validate:"required,eth_addr_checksum"`
		Amount       string `json:"amount" validate:"required,number,gt=0"`
	}

	AccountAddressParam struct {
		Address string `param:"address"  validate:"required,eth_addr_checksum"`
	}

	TrackingIDParam struct {
		TrackingID string `param:"trackingId"  validate:"required,uuid"`
	}

	OTXByAccountRequest struct {
		Address string `param:"address" validate:"required,eth_addr_checksum"`
		PerPage int    `query:"perPage" validate:"required,number,gt=0"`
		Cursor  int    `query:"cursor" validate:"number"`
		Next    bool   `query:"next"`
	}
)

const (
	ErrCodeInternalServerError = "E01"
	ErrCodeInvalidJSON         = "E02"
	ErrCodeInvalidAPIKey       = "E03"
	ErrCodeValidationFailed    = "E04"
	ErrCodeAccountNotExists    = "E05"
	ErrJWTAuth                 = "E06"
	ErrNoRecordFound           = "E07"
	ErrBannedToken             = "E08"
)
