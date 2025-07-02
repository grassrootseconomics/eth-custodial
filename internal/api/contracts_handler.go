package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

// ERC20Handler godoc
//
//	@Summary		ERC20 deploy request
//	@Description	ERC20 deploy request
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			transferRequest	body		apiresp.ERC20DeployRequest	true	"ERC20 deploy request"
//	@Success		200				{object}	apiresp.OKResponse
//	@Failure		400				{object}	apiresp.ErrResponse
//	@Failure		500				{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/contracts/erc20 [post]
func (a *API) contractsERC20Handler(c echo.Context) error {
	req := apiresp.ERC20DeployRequest{}

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

	trackingID := uuid.NewString()

	_, err = a.queueClient.InsertTx(c.Request().Context(), tx, worker.TokenDeployArgs{
		TrackingID:    trackingID,
		Name:          req.Name,
		Symbol:        req.Symbol,
		Decimals:      req.Decimals,
		InitialSupply: req.InitialSupply,
		InitialMintee: req.InitialMintee,
		Owner:         req.Owner,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "ERC20 deploy request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}
