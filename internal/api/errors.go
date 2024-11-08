package api

import (
	"errors"
	"net/http"

	"github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

func newBadRequestError(message ...interface{}) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusBadRequest, message...)
}

func (a *API) customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	if hErr, ok := err.(*echo.HTTPError); ok {
		var errorMsg string

		if m, ok := hErr.Message.(error); ok {
			errorMsg = m.Error()
		} else if m, ok := hErr.Message.(string); ok {
			errorMsg = m
		}

		c.JSON(hErr.Code, api.ErrResponse{
			Ok:          false,
			Description: errorMsg,
		})
		return
	}

	a.logg.Error("api: echo error", "path", c.Path(), "err", err)
	c.JSON(http.StatusInternalServerError, api.ErrResponse{
		Ok:          false,
		Description: "Internal server error",
		ErrCode:     api.ErrCodeInternalServerError,
	})
}

func handleBindError(c echo.Context) error {
	return c.JSON(http.StatusBadRequest, api.ErrResponse{
		Ok:          false,
		ErrCode:     api.ErrCodeInvalidJSON,
		Description: "Invalid or malformed request",
	})
}

func handleValidateError(c echo.Context) error {
	return c.JSON(http.StatusBadRequest, api.ErrResponse{
		Ok:          false,
		ErrCode:     api.ErrCodeValidationFailed,
		Description: "Validation failed on one or more fields",
	})
}

func handleJWTAuthError(c echo.Context, errorReason string) error {
	return c.JSON(http.StatusUnauthorized, api.ErrResponse{
		Ok:          false,
		ErrCode:     api.ErrJWTAuth,
		Description: "JWT authentication failed " + errorReason,
	})
}

func handlePostgresError(c echo.Context, err error) error {
	// TODO: Use a switch case to handle moree pg errors if needed
	if errors.Is(err, pgx.ErrNoRows) {
		return c.JSON(http.StatusNotFound, api.ErrResponse{
			Ok:          false,
			ErrCode:     api.ErrNoRecordFound,
			Description: "Record(s) not found",
		})
	}

	return err
}
