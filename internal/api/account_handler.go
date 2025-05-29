package api

import (
	"math/big"
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/grassrootseconomics/ethutils"
	"github.com/labstack/echo/v4"
	"github.com/lmittmann/w3/module/eth"
)

// accountCreateHandler godoc
//
//	@Summary		Create a new custodial account
//	@Description	Create a new custodial account
//	@Tags			Account
//	@Accept			*/*
//	@Produce		json
//	@Success		200	{object}	apiresp.OKResponse
//	@Failure		500	{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/account/create [post]
func (a *API) accountCreateHandler(c echo.Context) error {
	generatedKeyPair, err := keypair.GenerateKeyPair()
	if err != nil {
		return err
	}

	trackingID := uuid.NewString()

	_, err = a.queueClient.Insert(c.Request().Context(), worker.AccountCreateArgs{
		TrackingID: trackingID,
		KeyPair:    generatedKeyPair,
	}, nil)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Account creation request successfully created",
		Result: map[string]any{
			"publicKey":  generatedKeyPair.Public,
			"trackingId": trackingID,
		},
	})
}

// accountStatusHandler godoc
//
//	@Summary		Check a custodial account's status
//	@Description	Check a custodial account's status
//	@Tags			Account
//	@Accept			*/*
//	@Produce		json
//	@Param			address	path		string	true	"Account address"
//	@Success		200		{object}	apiresp.OKResponse
//	@Failure		500		{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/account/status/{address} [get]
func (a *API) accountStatusHandler(c echo.Context) error {
	req := apiresp.AccountAddressParam{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	var (
		gasBalance   *big.Int
		networkNonce uint64
	)

	accountAddress := ethutils.HexToAddress(req.Address)

	if err := a.chainProvider.Client.CallCtx(
		c.Request().Context(),
		eth.Nonce(accountAddress, nil).Returns(&networkNonce),
		eth.Balance(accountAddress, nil).Returns(&gasBalance),
	); err != nil {
		return err
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	internalNonce, err := a.store.PeekNonce(c.Request().Context(), tx, req.Address)
	if err != nil {
		return handlePostgresError(c, err)
	}

	active, err := a.store.CheckKeypair(c.Request().Context(), tx, req.Address)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Custodial account status",
		Result: map[string]any{
			"gasBalance":    gasBalance.String(),
			"networkNonce":  networkNonce,
			"internalNonce": internalNonce,
			"active":        active,
		},
	})
}
