package api

import (
	"net/http"

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
