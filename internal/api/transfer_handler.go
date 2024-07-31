package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
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

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	exists, err := a.store.CheckKeypair(c.Request().Context(), tx, req.From)
	if err != nil {
		return err
	}
	if !exists {
		return c.JSON(http.StatusNotFound, errResp{
			Ok:          false,
			Description: fmt.Sprintf("Account %s does not exist or is not yet activated", req.From),
		})
	}

	trackingId := uuid.NewString()

	_, err = a.queue.Client().InsertTx(c.Request().Context(), tx, worker.TokenTransferArgs{
		TrackingId:   trackingId,
		From:         req.From,
		To:           req.To,
		TokenAddress: req.TokenAddress,
		Amount:       req.Amount,
	}, nil)
	if err != nil {
		return err
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{
		Ok:          true,
		Description: "Transfer request successfully created",
		Result: H{
			"trackingId": trackingId,
		},
	})
}
