package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
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
//	@Router			/account/create [post]
func (a *API) accountCreateHandler(c echo.Context) error {
	generatedKeyPair, err := keypair.GenerateKeyPair()
	if err != nil {
		return err
	}

	trackingID := uuid.NewString()

	_, err = a.worker.QueueClient.Insert(c.Request().Context(), worker.AccountCreateArgs{
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

// systemInfoHandler godoc
//
//	@Summary		Get the current system information
//	@Description	Get the current system information
//	@Tags			System
//	@Accept			*/*
//	@Produce		json
//	@Success		200	{object}	apiresp.OKResponse
//	@Failure		500	{object}	apiresp.ErrResponse
//	@Router			/system [get]
func (a *API) systemInfoHandler(c echo.Context) error {
	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	systemKey, err := a.store.LoadMasterSignerKey(c.Request().Context(), tx)
	if err != nil {
		return err
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Current system information",
		Result: map[string]any{
			"systemSigner": systemKey.Public,
		},
	})
}
