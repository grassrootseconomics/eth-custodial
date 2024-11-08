package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

// transferHandler godoc
//
//	@Summary		Sign a token transfer request
//	@Description	Sign a token transfer request
//	@Tags			Sign
//	@Accept			json
//	@Produce		json
//	@Param			transferRequest	body		apiresp.TransferRequest	true	"Transfer request"
//	@Success		200				{object}	apiresp.OKResponse
//	@Failure		400				{object}	apiresp.ErrResponse
//	@Failure		500				{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/token/transfer [post]
func (a *API) transferHandler(c echo.Context) error {
	req := apiresp.TransferRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return handlePostgresError(c, err)
	}
	defer tx.Rollback(c.Request().Context())

	exists, err := a.store.CheckKeypair(c.Request().Context(), tx, req.From)
	if err != nil {
		return handlePostgresError(c, err)
	}
	if !exists {
		return c.JSON(http.StatusNotFound, apiresp.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Account %s does not exist or is not yet activated", req.From),
			ErrCode:     apiresp.ErrCodeAccountNotExists,
		})
	}

	trackingID := uuid.NewString()

	_, err = a.worker.QueueClient.InsertTx(c.Request().Context(), tx, worker.TokenTransferArgs{
		TrackingID:   trackingID,
		From:         req.From,
		To:           req.To,
		TokenAddress: req.TokenAddress,
		Amount:       req.Amount,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Transfer request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}
