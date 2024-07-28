package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (a *API) transferHandler(c echo.Context) error {
	var req struct {
		From         string `json:"from" validate:"required,eth_addr_checksum"`
		To           string `json:"to" validate:"required,eth_addr_checksum"`
		TokenAddress string `json:"tokenAddress" validate:"required,eth_addr_checksum"`
		Amount       string `json:"amount" validate:"number,gt=0"`
	}

	if err := c.Bind(&req); err != nil {
		return newBadRequestError(errInvalidJSON)
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{
		Ok: true,
		Result: H{
			"trackingId": 1,
		},
	})
}
