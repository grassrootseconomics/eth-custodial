package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

func (a *API) transferHandler(c echo.Context) error {
	req := api.TransferRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
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
		return c.JSON(http.StatusNotFound, api.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Account %s does not exist or is not yet activated", req.From),
			ErrCode:     api.ErrCodeAccountNotExists,
		})
	}

	trackingID := uuid.NewString()

	_, err = a.queue.Client().InsertTx(c.Request().Context(), tx, worker.TokenTransferArgs{
		TrackingId:   trackingID,
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

	return c.JSON(http.StatusOK, api.OKResponse{
		Ok:          true,
		Description: "Transfer request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}
