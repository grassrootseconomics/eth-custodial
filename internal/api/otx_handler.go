package api

import (
	"fmt"
	"net/http"

	"github.com/grassrootseconomics/eth-custodial/internal/store"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

// trackOTXHandler godoc
//
//	@Summary		Track an OTX's (Origin transaction) chain status
//	@Description	Track an OTX's (Origin transaction) chain status
//	@Tags			OTX
//	@Accept			*/*
//	@Produce		json
//	@Param			trackingId	path		string	true	"Tracking ID"
//	@Success		200			{object}	apiresp.OKResponse
//	@Failure		500			{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/otx/track/{trackingId} [get]
func (a *API) trackOTXHandler(c echo.Context) error {
	req := apiresp.TrackingIDParam{}

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

	otx, err := a.store.GetOTXByTrackingID(c.Request().Context(), tx, req.TrackingID)
	if err != nil {
		return err
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Current OTX chain status",
		Result: map[string]any{
			"otx": otx,
		},
	})
}

// getOTXByAddressHandler godoc
//
//	@Summary		Get an accounts OTX's (Origin transaction)
//	@Description	Get an accounts OTX's (Origin transaction)
//	@Tags			Account
//	@Accept			*/*
//	@Produce		json
//	@Param			address	path		string	true	"Account"
//	@Param			next	query		bool	false	"Next"
//	@Param			cursor	query		int		false	"Cursor"
//	@Param			perPage	query		int		true	"Per page"
//	@Success		200		{object}	apiresp.OKResponse
//	@Failure		500		{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/account/otx/{address} [get]
func (a *API) getOTXByAddressHandler(c echo.Context) error {
	req := apiresp.OTXByAccountRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		a.logg.Error("validation error", "error", err)
		return handleValidateError(c)
	}

	pagination := validatePagination(req)

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	var otx []*store.OTX

	if pagination.FirstPage {
		otx, err = a.store.GetOTXByAccount(c.Request().Context(), tx, req.Address, pagination.PerPage)
		if err != nil {
			return err
		}
	} else if pagination.Next {
		otx, err = a.store.GetOTXByAccountNext(c.Request().Context(), tx, req.Address, pagination.Cursor, pagination.PerPage)
		if err != nil {
			return err
		}
	} else {
		otx, err = a.store.GetOTXByAccountPrevious(c.Request().Context(), tx, req.Address, pagination.Cursor, pagination.PerPage)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return err
	}

	var first, last uint64

	if len(otx) > 0 {
		first = otx[0].ID
		last = otx[len(otx)-1].ID
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: fmt.Sprintf("Successfully fetched OTX for %s", req.Address),
		Result: map[string]any{
			"otx":   otx,
			"first": first,
			"last":  last,
		},
	})
}
