package api

import (
	"net/http"

	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

// systemInfoHandler godoc
//
//	@Summary		Get the current system information
//	@Description	Get the current system information
//	@Tags			System
//	@Accept			*/*
//	@Produce		json
//	@Success		200	{object}	apiresp.OKResponse
//	@Failure		500	{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
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
			"build":        a.build,
		},
	})
}
