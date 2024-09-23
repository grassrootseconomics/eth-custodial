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

	TransferRequest struct {
		From         string `json:"from" validate:"required,eth_addr_checksum"`
		To           string `json:"to" validate:"required,eth_addr_checksum"`
		TokenAddress string `json:"tokenAddress" validate:"required,eth_addr_checksum"`
		Amount       string `json:"amount" validate:"number,gt=0"`
	}
)

const (
	ErrCodeInternalServerError = "E01"
	ErrCodeInvalidJSON         = "E02"
	ErrCodeInvalidAPIKey       = "E03"
	ErrCodeValidationFailed    = "E04"
	ErrCodeAccountNotExists    = "E05"
)
